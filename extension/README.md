# EasyEDA Connector Extension

This directory will contain the thin EasyEDA connector extension.

The connector should remain deliberately small:

- connect to the Go daemon
- register the EasyEDA window
- translate typed actions to official `eda.*` calls
- normalize results
- convert `File`/`Blob` values into artifacts
- return structured errors

It should not contain business workflows. Those belong in the Go action layer and Skill instructions.
