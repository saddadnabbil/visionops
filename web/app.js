const $ = (selector, root = document) => root.querySelector(selector);
const escapeHTML = (value = "") => String(value).replace(/[&<>'"]/g, character => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", "'": "&#39;", '"': "&quot;" })[character]);
const formatDate = value => value && value !== "never" ? new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(new Date(value)) : "Never";
const titleCase = value => String(value || "").replaceAll("_", " ").replace(/\b\w/g, letter => letter.toUpperCase());

const roleRoutes = {
  admin: ["overview", "incidents", "cameras", "delivery", "analytics", "organization"],
  operator: ["incidents", "cameras", "delivery"],
  supervisor: ["analytics", "incidents", "cameras", "delivery"],
  viewer: ["overview", "incidents", "cameras"]
};
const routeLabels = { overview: "Overview", incidents: "Incidents", cameras: "Cameras", delivery: "Delivery", analytics: "Analytics", organization: "Organization" };
const routeEyebrows = { overview: "SAFETY OPERATIONS", incidents: "LIVE QUEUE", cameras: "OBSERVATION POINTS", delivery: "DURABLE DELIVERY", analytics: "SAFETY PERFORMANCE", organization: "ADMINISTRATION" };
const roleHome = { admin: "overview", operator: "incidents", supervisor: "analytics", viewer: "overview" };

const state = {
  token: sessionStorage.getItem("visionops_token") || "", profile: null,
  incidents: [], cameras: [], jobs: [], deliveries: [], operations: null, observability: null,
  failureMode: false, eventSource: null, refreshTimer: null, loading: false
};

async function api(path, options = {}) {
  const response = await fetch(`/api/v1${path}`, { ...options, headers: { "Content-Type": "application/json", ...(state.token ? { Authorization: `Bearer ${state.token}` } : {}), ...(options.headers || {}) } });
  const body = response.status === 204 ? null : await response.json().catch(() => ({}));
  if (!response.ok) { const error = new Error(body.error || `Request failed (${response.status})`); error.status = response.status; throw error; }
  return body;
}

function notify(message, tone = "neutral") {
  const toast = $("#toast"); toast.textContent = message; toast.dataset.tone = tone; toast.hidden = false;
  clearTimeout(notify.timer); notify.timer = setTimeout(() => { toast.hidden = true; }, 3200);
}
function showLanding() {
  closeLiveUpdates(); $("#app").hidden = true; $("#login-view").hidden = true; $("#landing-view").hidden = false;
}
function showLogin(message = "") {
  closeLiveUpdates(); $("#app").hidden = true; $("#landing-view").hidden = true; $("#login-view").hidden = false;
  const error = $("#login-error"); error.textContent = message; error.hidden = !message; $("#email").focus();
}
async function signIn(email, password) {
  const response = await fetch("/api/v1/auth/login", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ email, password }) });
  const body = await response.json().catch(() => ({})); if (!response.ok) throw new Error(body.error || "Unable to sign in");
  state.token = body.token; sessionStorage.setItem("visionops_token", state.token); state.profile = await api("/auth/me"); history.replaceState(null, "", "/"); showApp();
}
function signOut() { sessionStorage.removeItem("visionops_token"); state.token = ""; state.profile = null; history.replaceState(null, "", location.pathname); showLanding(); }
function showApp() {
  $("#landing-view").hidden = true; $("#login-view").hidden = true; $("#app").hidden = false;
  $("#organization-name").textContent = state.profile.organization; $("#user-email").textContent = state.profile.email;
  $("#header-role").textContent = titleCase(state.profile.role); $("#sidebar-role").textContent = titleCase(state.profile.role); renderNavigation();
  const requested = currentRoute(); const destination = allowedRoutes().includes(requested) ? requested : roleHome[state.profile.role];
  if (requested !== destination) location.hash = `#/${destination}`; connectLiveUpdates(); renderRoute();
}

function allowedRoutes() { return roleRoutes[state.profile?.role] || []; }
function currentRoute() { return location.hash.replace(/^#\//, "").split("/")[0] || roleHome[state.profile?.role] || "overview"; }
function canRespond() { return ["admin", "operator"].includes(state.profile?.role); }
function isAdmin() { return state.profile?.role === "admin"; }
function renderNavigation() { $("#primary-nav").innerHTML = allowedRoutes().map(route => `<a href="#/${route}" data-route="${route}">${routeLabels[route]}</a>`).join(""); syncActiveNavigation(); }
function syncActiveNavigation() { document.querySelectorAll("[data-route]").forEach(link => link.toggleAttribute("aria-current", link.dataset.route === currentRoute())); }
function closeNavigation() { $("#sidebar").dataset.open = "false"; $("#menu-toggle").setAttribute("aria-expanded", "false"); }

function pageHeader(title, intro, action = "") { return `<header class="page-header"><div><p class="eyebrow">${routeEyebrows[currentRoute()]}</p><h1>${title}</h1><p>${intro}</p></div>${action ? `<div class="page-actions">${action}</div>` : ""}</header>`; }
function loadingView() { return `<section class="state-panel loading-shell" aria-live="polite" aria-busy="true"><p class="eyebrow">PREPARING WORKSPACE</p><div class="loading-lines" aria-hidden="true"><span></span><span></span><span></span></div><h2>Getting operations ready.</h2><p>Loading only the information this view needs.</p></section>`; }
function errorView(error) { return `<div class="state-panel error-state"><p class="eyebrow">COULD NOT LOAD</p><h2>Operations are temporarily unavailable.</h2><p>${escapeHTML(error.message)}</p><button class="button primary" data-action="retry">Try again</button></div>`; }
function statusBadge(status) { return `<span class="status-badge" data-status="${escapeHTML(status)}"><span aria-hidden="true"></span>${escapeHTML(titleCase(status))}</span>`; }
function selectMenu({ id, name = "", label, options, value = "", filter = false }) { const selected = options.find(option => option.value === value) || options[0]; return `<div class="select-menu" data-select${filter ? " data-incident-filter" : ""}><input type="hidden" ${name ? `name="${escapeHTML(name)}"` : ""} value="${escapeHTML(selected.value)}"><button id="${escapeHTML(id)}" class="select-trigger" type="button" aria-haspopup="listbox" aria-expanded="false"><span>${escapeHTML(selected.label)}</span><span class="select-chevron" aria-hidden="true">⌄</span></button><div class="select-options" role="listbox" aria-labelledby="${escapeHTML(id)}" hidden>${options.map(option => `<button type="button" role="option" aria-selected="${option.value === selected.value}" data-select-option="${escapeHTML(option.value)}">${escapeHTML(option.label)}</button>`).join("")}</div></div>`; }
function filterIncidents(value) { $("#incident-list").innerHTML = state.incidents.filter(item => value === "all" || item.status === value).map(incidentRow).join("") || `<div class="empty-state"><h3>No matching incidents</h3><p>Choose another queue state.</p></div>`; }
async function loadCore(route) {
  if (route === "overview") {
    const [incidents, cameras, jobs] = await Promise.all([api("/incidents?limit=100"), api("/cameras"), api("/jobs")]);
    Object.assign(state, { incidents, cameras, jobs });
    return;
  }
  if (route === "incidents") { state.incidents = await api("/incidents?limit=100"); return; }
  if (route === "cameras") { state.cameras = await api("/cameras"); return; }
  if (route === "delivery") {
    const [jobs, deliveries, observability] = await Promise.all([api("/jobs"), api("/deliveries"), api("/metrics/observability")]);
    Object.assign(state, { jobs, deliveries, observability });
    return;
  }
  if (route === "analytics") {
    const [operations, observability] = await Promise.all([api("/metrics/operations"), api("/metrics/observability")]);
    Object.assign(state, { operations, observability });
  }
}
async function renderRoute(options = {}) {
  if (!state.profile) return; const route = currentRoute();
  if (!allowedRoutes().includes(route)) { location.hash = `#/${roleHome[state.profile.role]}`; return; }
  if (!options.silent) closeNavigation();
  syncActiveNavigation(); const page = $("#page");
  // Keep an already-rendered route visible during navigation. The initial shell
  // communicates progress without a spinner that can look frozen on a slow request.
  if (!options.silent && !page.childElementCount) page.innerHTML = loadingView();
  try {
    state.loading = true; await loadCore(route);
    const renderers = { overview: renderOverview, incidents: renderIncidents, cameras: renderCameras, delivery: renderDelivery, analytics: renderAnalytics, organization: renderOrganization };
    await renderers[route]();
    if (!options.silent) window.scrollTo({ top: 0, behavior: "auto" });
    page.focus({ preventScroll: true });
  } catch (error) { if (error.status === 401) return signOut(); page.innerHTML = errorView(error); }
  finally { state.loading = false; }
}

function metricCard(label, value, note) { return `<article class="metric-card"><span class="eyebrow">${label}</span><strong>${value}</strong><p>${note}</p></article>`; }
function incidentRow(incident) { return `<button class="incident-row" data-incident-id="${escapeHTML(incident.id)}"><span class="incident-main"><span class="severity" data-severity="${escapeHTML(incident.severity)}">${escapeHTML(incident.severity)}</span><strong>${escapeHTML(titleCase(incident.title))}</strong><small>${escapeHTML(incident.camera.name)} · ${incident.occurrences} detection${incident.occurrences === 1 ? "" : "s"}</small></span><span class="incident-meta">${statusBadge(incident.status)}<time>${formatDate(incident.last_seen_at)}</time><span class="row-arrow" aria-hidden="true">→</span></span></button>`; }
function renderOverview() {
  const open = state.incidents.filter(incident => incident.status !== "resolved"); const critical = open.filter(incident => incident.severity === "critical").length;
  const online = state.cameras.filter(camera => camera.status === "online").length; const dead = state.jobs.filter(job => job.status === "dead").length;
  const title = state.profile.role === "admin" ? "Keep operations accountable." : "See what needs attention.";
  $("#page").innerHTML = `${pageHeader(title, state.profile.role === "admin" ? "Configuration, safety response, and delivery health in one workspace." : "Read-only visibility into current safety operations.", canRespond() ? `<button class="button primary" data-action="simulate">Simulate detection</button>` : "")}
    <section class="metric-grid" aria-label="Operational summary">${metricCard("OPEN INCIDENTS", open.length, critical ? `${critical} critical` : "No critical incidents")}${metricCard("CAMERAS ONLINE", `${online}/${state.cameras.length}`, "Heartbeat-derived health")}${metricCard("DEAD DELIVERIES", dead, dead ? "Recovery action required" : "Delivery queue healthy")}</section>
    <section class="story-block lime-block"><div><p class="eyebrow">ATTENTION QUEUE</p><h2>${open.length ? `${open.length} incident${open.length === 1 ? "" : "s"}, ready for action.` : "The queue is clear."}</h2></div><p>Detections become tenant-scoped incidents with an activity trail and durable notification delivery.</p></section>
    <section class="content-section"><div class="section-title"><div><p class="eyebrow">LATEST ACTIVITY</p><h2>Recent incidents</h2></div><a class="text-link" href="#/incidents">View complete queue →</a></div><div class="incident-list">${state.incidents.slice(0, 5).map(incidentRow).join("") || `<div class="empty-state"><h3>No incidents yet</h3><p>${canRespond() ? "Simulate a detection to exercise the complete workflow." : "New detections will appear here when they arrive."}</p></div>`}</div></section>`;
}
function renderIncidents() {
  $("#page").innerHTML = `${pageHeader("Incidents, ready for action.", canRespond() ? "Triage the queue, take ownership, and leave an auditable resolution." : "Review the live queue and activity history with read-only access.", canRespond() ? `<button class="button primary" data-action="simulate">Simulate detection</button>` : "")}
    <section class="story-block lime-block compact-story"><div><p class="eyebrow">LIVE OPERATIONS</p><h2>${state.incidents.filter(item => item.status !== "resolved").length} unresolved</h2></div><div class="filter-control"><span>Show</span>${selectMenu({ id: "incident-filter", options: [{ value: "all", label: "All incidents" }, { value: "open", label: "Open incidents" }, { value: "acknowledged", label: "Acknowledged" }, { value: "resolved", label: "Resolved" }], filter: true })}</div></section>
    <section class="content-section"><div id="incident-list" class="incident-list">${state.incidents.map(incidentRow).join("") || `<div class="empty-state"><h3>No incidents yet</h3><p>Incoming safety detections will be correlated here.</p></div>`}</div></section>`;
}
function cameraCard(camera) { return `<article class="data-card"><div class="card-top">${statusBadge(camera.status)}<span class="mono-label">${escapeHTML(camera.id.slice(0, 8))}</span></div><h3>${escapeHTML(camera.name)}</h3><p>${escapeHTML(camera.location)}</p><dl><div><dt>Last detection</dt><dd>${formatDate(camera.last_detection)}</dd></div></dl></article>`; }
function recordedDemoCard() { return `<section class="recorded-demo" aria-labelledby="recorded-demo-title"><div class="recorded-demo-copy"><p class="eyebrow">RECORDED SCENARIO</p><h2 id="recorded-demo-title">Warehouse-style worksite demo.</h2><p>This is licensed recorded footage, not live CCTV. A reviewed timestamp scenario can create a real missing-hard-hat incident through the adapter; no face recognition or video-model inference occurs.</p><dl><div><dt>Source</dt><dd>Mixkit construction-site footage</dd></div><div><dt>Event</dt><dd>00:02 · missing hard hat</dd></div></dl></div><figure><video controls muted playsinline preload="metadata" aria-label="Recorded worksite scenario preview"><source src="/assets/demo/construction-site-wide-recorded-scenario.mp4" type="video/mp4">Your browser cannot play this recorded demo video.</video><figcaption>RECORDED SCENARIO / SIMULATED DETECTOR — NOT LIVE CCTV</figcaption></figure></section>`; }
function renderCameras() {
  $("#page").innerHTML = `${pageHeader("Know every observation point.", "Heartbeat health tells the team whether a registered camera service is still communicating.", isAdmin() ? `<button class="button primary" data-action="toggle-camera-form">Add camera</button>` : "")}
    ${isAdmin() ? `<form id="camera-form" class="inline-form" hidden><div><p class="eyebrow">REGISTER CAMERA</p><h2>New observation point</h2></div><label>Name<input name="name" required placeholder="Line B Entrance"></label><label>Location<input name="location" required placeholder="Factory floor — Line B"></label><div class="form-actions"><button type="button" class="button secondary" data-action="cancel-camera">Cancel</button><button class="button primary" type="submit">Create camera</button></div></form>` : ""}
    ${recordedDemoCard()}
    <section class="card-grid">${state.cameras.map(cameraCard).join("") || `<div class="empty-state"><h3>No cameras registered</h3><p>An Admin must register an observation point before ingest.</p></div>`}</section>`;
}
function jobRow(job) { const replay = job.status === "dead" && canRespond() ? `<button class="button secondary compact" data-replay-id="${escapeHTML(job.id)}">Replay</button>` : ""; return `<article class="operation-row"><div>${statusBadge(job.status)}<h3>${escapeHTML(job.topic)}</h3><p>${job.attempts} attempt${job.attempts === 1 ? "" : "s"}${job.last_error ? ` · ${escapeHTML(job.last_error)}` : ""}</p></div><div><time>${formatDate(job.available_at)}</time>${replay}</div></article>`; }
function renderDelivery() {
  const pending = state.jobs.filter(job => job.status !== "done");
  $("#page").innerHTML = `${pageHeader("Every alert, accounted for.", "Inspect retries, dead-letter jobs, and the delivery attempts retained for audit.", isAdmin() ? `<button class="button ${state.failureMode ? "danger" : "secondary"}" data-action="failure-mode">${state.failureMode ? "Stop failure simulation" : "Simulate webhook failure"}</button>` : "")}
    <section class="metric-grid two">${metricCard("ACTIVE JOBS", pending.length, "Pending, processing, or dead")}${metricCard("DELIVERY ATTEMPTS", state.deliveries.length, `${state.observability.webhook_failures} recorded failures`)}</section>
    <section class="content-section"><div class="section-title"><div><p class="eyebrow">OUTBOX</p><h2>Delivery jobs</h2></div></div><div class="operation-list">${pending.map(jobRow).join("") || `<div class="empty-state"><h3>No recovery work</h3><p>Every queued alert has completed delivery.</p></div>`}</div></section>`;
}
function renderAnalytics() {
  const severity = state.operations.open_by_severity || {}; const recurring = state.operations.recurring_rules || [];
  $("#page").innerHTML = `${pageHeader("See the safety pattern.", "Measure response time, recurring rules, and system throughput without changing live incidents.")}
    <section class="metric-grid">${metricCard("AVG ACKNOWLEDGE", `${Math.round(state.operations.average_ack_minutes || 0)}m`, "From first detection")}${metricCard("AVG RESOLUTION", `${Math.round(state.operations.average_resolution_minutes || 0)}m`, "From first detection")}${metricCard("TOTAL DETECTIONS", state.observability.detections, `${state.observability.incidents} correlated incidents`)}</section>
    <section class="analytics-grid"><article class="story-block lilac-block vertical"><div><p class="eyebrow">OPEN BY SEVERITY</p><h2>${Object.values(severity).reduce((sum, count) => sum + count, 0)} unresolved</h2></div><div class="breakdown">${["critical", "high", "medium", "low"].map(level => `<div><span>${titleCase(level)}</span><strong>${severity[level] || 0}</strong></div>`).join("")}</div></article><article class="data-card large"><p class="eyebrow">RECURRING RULES</p><h2>Where risk repeats</h2><ol class="ranking">${recurring.map(item => `<li><span>${escapeHTML(titleCase(item.rule))}</span><strong>${item.count}</strong></li>`).join("") || `<li><span>No incident history</span><strong>0</strong></li>`}</ol></article></section>`;
}
async function renderOrganization() {
  const [users, keys, webhooks] = await Promise.all([api("/users"), api("/api-keys"), api("/webhooks")]);
  $("#page").innerHTML = `${pageHeader("Configure the workspace.", "Manage people, ingestion credentials, webhook destinations, and their tenant boundary.")}
    <section class="admin-grid"><article class="admin-panel"><div class="section-title"><div><p class="eyebrow">PEOPLE</p><h2>Users</h2></div><button class="button secondary compact" data-action="toggle-user-form">Add user</button></div><form id="user-form" class="stack-form" hidden><label>Email<input name="email" type="email" required></label><label>Temporary password<input name="password" type="password" minlength="12" required></label><label>Role${selectMenu({ id: "user-role", name: "role", value: "operator", options: [{ value: "operator", label: "Operator" }, { value: "supervisor", label: "Supervisor" }, { value: "viewer", label: "Viewer" }, { value: "admin", label: "Admin" }] })}</label><button class="button primary" type="submit">Create user</button></form><div class="simple-list">${users.map(user => `<div><span><strong>${escapeHTML(user.email)}</strong><small>${titleCase(user.role)}</small></span><time>${formatDate(user.created_at)}</time></div>`).join("")}</div></article>
      <article class="admin-panel"><div class="section-title"><div><p class="eyebrow">INGESTION</p><h2>API keys</h2></div><button class="button secondary compact" data-action="toggle-key-form">Create key</button></div><form id="key-form" class="stack-form" hidden><label>Key name<input name="name" required placeholder="Production detector"></label><button class="button primary" type="submit">Create one-time key</button></form><div id="new-key"></div><div class="simple-list">${keys.map(key => `<div><span><strong>${escapeHTML(key.name)}</strong><small>${key.active ? "Active" : "Inactive"}</small></span><time>${formatDate(key.created_at)}</time></div>`).join("")}</div></article>
      <article class="admin-panel wide"><div class="section-title"><div><p class="eyebrow">DESTINATIONS</p><h2>Webhook subscriptions</h2></div><button class="button secondary compact" data-action="toggle-webhook-form">Add webhook</button></div><form id="webhook-form" class="stack-form horizontal" hidden><label>HTTPS URL<input name="url" type="url" required placeholder="https://alerts.example.test/visionops"></label><label>Signing secret<input name="secret" minlength="12" required></label><button class="button primary" type="submit">Create subscription</button></form><div class="simple-list">${webhooks.map(item => `<div><span><strong>${escapeHTML(item.url)}</strong><small>${item.enabled ? "Enabled" : "Disabled"}</small></span><time>${formatDate(item.created_at)}</time></div>`).join("")}</div></article></section>`;
}

async function showIncident(id) {
  try {
    const incident = await api(`/incidents/${id}`);
    const actions = canRespond() && incident.status !== "resolved" ? `<form id="incident-action-form" class="resolution-form"><label>Operator note<textarea name="note" placeholder="Record what happened and what action was taken"></textarea></label><div class="form-actions">${incident.status === "open" ? `<button class="button secondary" type="button" data-incident-action="acknowledge" data-id="${incident.id}">Acknowledge</button>` : ""}<button class="button primary" type="button" data-incident-action="resolve" data-id="${incident.id}">Resolve incident</button></div></form>` : `<div class="read-only-note"><strong>${incident.status === "resolved" ? "Resolution" : "Read-only access"}</strong><p>${escapeHTML(incident.resolution_note || (canRespond() ? "No resolution note." : "Your role can review this incident but cannot change it."))}</p></div>`;
    $("#detail-body").innerHTML = `<div class="dialog-head"><div><p class="eyebrow">INCIDENT DETAIL</p><h2 id="detail-title">${escapeHTML(titleCase(incident.title))}</h2></div><button class="icon-button" data-action="close-detail" aria-label="Close incident detail">×</button></div><div class="detail-summary">${statusBadge(incident.status)}<span class="severity" data-severity="${incident.severity}">${escapeHTML(incident.severity)}</span><p>${escapeHTML(incident.camera.name)} · ${escapeHTML(incident.camera.location)}</p><dl><div><dt>Occurrences</dt><dd>${incident.occurrences}</dd></div><div><dt>First seen</dt><dd>${formatDate(incident.first_seen_at)}</dd></div><div><dt>Last seen</dt><dd>${formatDate(incident.last_seen_at)}</dd></div></dl></div><section class="timeline"><p class="eyebrow">ACTIVITY</p><ol>${incident.activity.map(activity => `<li><span></span><div><strong>${escapeHTML(titleCase(activity.type))}</strong><p>${escapeHTML(activity.actor)}${activity.note ? ` · ${escapeHTML(activity.note)}` : ""}</p><time>${formatDate(activity.created_at)}</time></div></li>`).join("")}</ol></section>${actions}`;
    $("#incident-dialog").showModal();
    document.body.classList.add("modal-open");
  } catch (error) { notify(error.message, "error"); }
}
async function simulateDetection() {
  try { await api("/demo/detections", { method: "POST" }); notify("Detection accepted and queued for delivery.", "success"); await renderRoute({ silent: true }); }
  catch (error) { notify(error.message, "error"); }
}
async function mutateIncident(id, action) {
  const note = new FormData($("#incident-action-form")).get("note") || "";
  if (action === "resolve" && !String(note).trim()) return notify("Add a resolution note before resolving.", "error");
  try { await api(`/incidents/${id}/${action}`, { method: "POST", body: JSON.stringify({ note }) }); $("#incident-dialog").close(); notify(action === "resolve" ? "Incident resolved." : "Incident acknowledged.", "success"); await renderRoute({ silent: true }); }
  catch (error) { notify(error.message, "error"); }
}
async function handlePageClick(event) {
  const option = event.target.closest("[data-select-option]");
  if (option) { const menu = option.closest("[data-select]"); menu.querySelector("input").value = option.dataset.selectOption; menu.querySelector(".select-trigger span").textContent = option.textContent; menu.querySelectorAll("[data-select-option]").forEach(item => item.setAttribute("aria-selected", String(item === option))); menu.querySelector(".select-options").hidden = true; menu.querySelector(".select-trigger").setAttribute("aria-expanded", "false"); if (menu.hasAttribute("data-incident-filter")) filterIncidents(option.dataset.selectOption); return; }
  const trigger = event.target.closest(".select-trigger");
  if (trigger) { const menu = trigger.closest("[data-select]"); const options = menu.querySelector(".select-options"); const isOpen = !options.hidden; document.querySelectorAll(".select-options").forEach(item => item.hidden = true); document.querySelectorAll(".select-trigger").forEach(item => item.setAttribute("aria-expanded", "false")); options.hidden = isOpen; trigger.setAttribute("aria-expanded", String(!isOpen)); return; }
  const incident = event.target.closest("[data-incident-id]"); if (incident) return showIncident(incident.dataset.incidentId);
  const replay = event.target.closest("[data-replay-id]");
  if (replay) { try { await api(`/jobs/${replay.dataset.replayId}/replay`, { method: "POST" }); notify("Dead job queued for replay.", "success"); await renderRoute({ silent: true }); } catch (error) { notify(error.message, "error"); } return; }
  const action = event.target.closest("[data-action]")?.dataset.action; if (!action) return;
  if (action === "retry") return renderRoute(); if (action === "simulate") return simulateDetection();
  if (action === "toggle-camera-form") return $("#camera-form").hidden = false; if (action === "cancel-camera") return $("#camera-form").hidden = true;
  if (action.startsWith("toggle-")) { const form = $(`#${action.replace("toggle-", "")}`); if (form) form.hidden = !form.hidden; }
  if (action === "failure-mode") { try { state.failureMode = !state.failureMode; await api("/demo/failure-mode", { method: "POST", body: JSON.stringify({ enabled: state.failureMode }) }); notify(state.failureMode ? "Webhook failure simulation enabled." : "Webhook failure simulation stopped.", "success"); await renderRoute({ silent: true }); } catch (error) { state.failureMode = !state.failureMode; notify(error.message, "error"); } }
}
async function handlePageSubmit(event) {
  event.preventDefault(); const data = Object.fromEntries(new FormData(event.target));
  try {
    if (event.target.id === "camera-form") await api("/cameras", { method: "POST", body: JSON.stringify(data) });
    if (event.target.id === "user-form") await api("/users", { method: "POST", body: JSON.stringify(data) });
    if (event.target.id === "webhook-form") await api("/webhooks", { method: "POST", body: JSON.stringify(data) });
    if (event.target.id === "key-form") { const result = await api("/api-keys", { method: "POST", body: JSON.stringify(data) }); $("#new-key").innerHTML = `<div class="secret-callout"><strong>Copy this key now</strong><code>${escapeHTML(result.api_key)}</code><p>It cannot be retrieved again.</p></div>`; notify("API key created.", "success"); return; }
    notify("Configuration saved.", "success"); await renderRoute({ silent: true });
  } catch (error) { notify(error.message, "error"); }
}
function connectLiveUpdates() {
  closeLiveUpdates(); const connection = $("#connection"); connection.textContent = "Connecting"; connection.dataset.state = "connecting";
  state.eventSource = new EventSource(`/api/v1/events?token=${encodeURIComponent(state.token)}`);
  state.eventSource.addEventListener("connected", () => { connection.textContent = "Live"; connection.dataset.state = "live"; });
  state.eventSource.addEventListener("update", () => { if (shouldRefreshLiveData()) renderRoute({ silent: true }); });
  state.eventSource.onerror = () => { connection.textContent = "Updates paused"; connection.dataset.state = "paused"; };
  state.refreshTimer = setInterval(() => { if (!document.hidden && shouldRefreshLiveData()) renderRoute({ silent: true }); }, 30000);
}
function closeLiveUpdates() { state.eventSource?.close(); clearInterval(state.refreshTimer); state.eventSource = null; }
function shouldRefreshLiveData() {
  return !state.loading && currentRoute() !== "organization" && !document.querySelector("#camera-form:not([hidden]), .stack-form:not([hidden])");
}

$("#login-form").addEventListener("submit", async event => { event.preventDefault(); const button = event.submitter; button.disabled = true; button.textContent = "Signing in…"; try { await signIn($("#email").value, $("#password").value); } catch (error) { showLogin(error.message); } finally { button.disabled = false; button.textContent = "Sign in"; } });
$("#demo-role-buttons").innerHTML = ["admin", "operator", "supervisor", "viewer"].map(role => `<button type="button" data-demo-role="${role}">${titleCase(role)}</button>`).join("");
$("#demo-role-buttons").addEventListener("click", event => { const role = event.target.dataset.demoRole; if (role) { $("#email").value = `${role}@acme.test`; $("#password").value = "demo-password"; } });
$("#logout").addEventListener("click", signOut);
$("#menu-toggle").addEventListener("click", () => { const open = $("#sidebar").dataset.open !== "true"; $("#sidebar").dataset.open = String(open); $("#menu-toggle").setAttribute("aria-expanded", String(open)); });
$("#nav-scrim").addEventListener("click", closeNavigation); $("#primary-nav").addEventListener("click", closeNavigation); $("#page").addEventListener("click", handlePageClick); $("#page").addEventListener("submit", handlePageSubmit);
document.addEventListener("click", event => { if (!event.target.closest("[data-select]")) { document.querySelectorAll(".select-options").forEach(item => item.hidden = true); document.querySelectorAll(".select-trigger").forEach(item => item.setAttribute("aria-expanded", "false")); } });
document.addEventListener("keydown", event => { if (event.key === "Escape") { document.querySelectorAll(".select-options").forEach(item => item.hidden = true); document.querySelectorAll(".select-trigger").forEach(item => item.setAttribute("aria-expanded", "false")); } });
$("#incident-dialog").addEventListener("click", event => {
  const dialog = $("#incident-dialog");
  // A click on the native dialog backdrop is retargeted to the dialog itself.
  if (event.target === dialog || event.target.dataset.action === "close-detail") dialog.close();
  const action = event.target.dataset.incidentAction;
  if (action) mutateIncident(event.target.dataset.id, action);
});
$("#incident-dialog").addEventListener("close", () => document.body.classList.remove("modal-open"));
window.addEventListener("hashchange", () => { if (!state.profile) return location.pathname === "/login" ? showLogin() : showLanding(); renderRoute(); });
(async function boot() { if (!state.token) return location.pathname === "/login" ? showLogin() : showLanding(); try { state.profile = await api("/auth/me"); showApp(); } catch { signOut(); } })();
