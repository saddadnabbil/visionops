# VisionOps Safety Operations

VisionOps is a workplace-safety operations context. It turns machine-produced safety signals into incidents that human operators can manage and audit.

## Language

**Detection**:
A raw observation from an AI camera or another approved safety-analysis producer. A Detection is evidence, not an operator task.
_Avoid_: Alert, incident

**Incident**:
The operational safety case created from one or more correlated Detections. It remains open, acknowledged, or resolved until its workflow is complete.
_Avoid_: Detection, violation

**Safety Rule**:
A named condition a producer evaluates, such as missing PPE, restricted-area entry, or crowding.
_Avoid_: Model, alert type

**Camera**:
A registered observation point associated with a physical location. It identifies the source location; it is not the AI analysis service itself.
_Avoid_: Detector, incident source

**Safety Operator**:
A person responsible for triaging an Incident and recording acknowledgement or resolution.
_Avoid_: Viewer, responder

**Safety Supervisor**:
A person accountable for safety operations who reviews incidents, escalation patterns, and team response.
_Avoid_: Administrator, operator

**AI Camera Service**:
An external producer that obtains a camera stream, evaluates Safety Rules, and submits Detections to VisionOps. It is outside the VisionOps application boundary.
_Avoid_: VisionOps camera, VisionOps model

**Fixture Camera Adapter**:
An approved synthetic producer that exercises the same Camera discovery, heartbeat, and Detection contract without accessing video or performing inference. It is used for local integration demonstrations.
_Avoid_: AI camera, CCTV feed

**Webhook Subscription**:
An approved destination that receives a signed notification about an Incident.
_Avoid_: Alert channel, integration
