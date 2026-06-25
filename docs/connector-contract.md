# EasyEDA Connector Contract

The connector is an EasyEDA extension installed in the official client. It is the only part of the system that can access the official `eda` object.

## Startup

1. Connect to the Go daemon by scanning `127.0.0.1:49620-49629`.
2. Verify the daemon service identity as `easyeda-agent`.
3. Register a generated `windowId`.
4. Send current context if a project/document is active.
5. Keep heartbeat active.

## Required Connector Messages

### register

```json
{
  "type": "register",
  "windowId": "uuid",
  "connectorVersion": "0.1.0",
  "easyedaVersion": "3.x",
  "capabilities": ["schematic.v1"]
}
```

### context

```json
{
  "type": "context",
  "windowId": "uuid",
  "projectUuid": "...",
  "documentUuid": "...",
  "documentType": "schematic"
}
```

### action result

Use the response envelope defined in [protocol.md](protocol.md).

## Official API Mapping for Phase 1

The connector will map actions to these EasyEDA APIs first:

- `eda.dmt_Project.getCurrentProjectInfo`
- `eda.dmt_SelectControl.getCurrentDocumentInfo`
- `eda.dmt_Schematic.getAllSchematicsInfo`
- `eda.dmt_Schematic.getAllSchematicPagesInfo`
- `eda.dmt_EditorControl.openDocument`
- `eda.dmt_EditorControl.getCurrentRenderedAreaImage`
- `eda.sch_PrimitiveComponent.getAll`
- `eda.sch_PrimitiveComponent.create`
- `eda.sch_PrimitiveComponent.modify`
- `eda.sch_PrimitiveComponent.delete`
- `eda.sch_PrimitiveWire.create`
- `eda.sch_PrimitiveComponent.createNetFlag`
- `eda.sch_PrimitiveComponent.createNetPort`
- `eda.sch_SelectControl.doSelectPrimitives`
- `eda.sch_SelectControl.getSelectedPrimitives_PrimitiveId`
- `eda.sch_Drc.check`
- `eda.sch_Document.save`
- `eda.sch_ManufactureData.getNetlistFile`
- `eda.sch_ManufactureData.getBomFile`

## Serialization Rules

- Convert primitive objects to plain JSON before returning.
- Include primitive ID, type, common coordinates, designator, net, and pins when available.
- Convert `File` and `Blob` to artifacts, not inline JSON.
- Preserve the original EasyEDA error message in `error.detail`.
- Add a stable `error.code` chosen by the connector or daemon.
