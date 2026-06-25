// Live orientation calibration — ground-truth the body-direction anchors that
// orientation.json (and tests/run.py) assert structurally. Run via debug.exec_js
// against a CONNECTED EasyEDA window, ideally after importing a new .eext:
//
//   easyeda exec --window <id> --file tools/schematic-lint/calibrate.js
//   # or pipe through the daemon /action like lint.sh does.
//
// For each family it creates a flag at every rotation, measures the body
// direction from the bbox-center offset (pure data, no screenshot), deletes it,
// then checks that rot 0 lands on the expected anchor and that the cycle is the
// expected up→left→down→right. Mutates the schematic only transiently (creates
// then deletes at a far-off scratch point). Returns a PASS/FAIL report.
//
// The EXPECTED facts below MUST stay byte-identical to orientation.json. If a
// new .eext changes them, update BOTH this file and orientation.json, then
// re-run tools/schematic-lint/tests/run.py --update.
const EXPECTED = {
  rotationCycle: ['up', 'left', 'down', 'right'],
  bodyAnchorAtRot0: { power: 'up', ground: 'down', port: 'right' },
};

const SCRATCH = { x: 9000, y: 9000 };   // far from any real circuit
const ROTATIONS = [0, 90, 180, 270];
const FAMILY_KIND = { power: 'Power', ground: 'Ground', port: 'OUT' };

function bboxCenter(bb) {
  if (!bb) return null;
  if (Array.isArray(bb)) {                       // [x0,y0,x1,y1]
    return { x: (bb[0] + bb[2]) / 2, y: (bb[1] + bb[3]) / 2 };
  }
  if ('width' in bb || 'minX' in bb) {           // {x,y,width,height} or {minX,...}
    const x0 = bb.x ?? bb.minX, y0 = bb.y ?? bb.minY;
    const w = bb.width ?? (bb.maxX - bb.minX), h = bb.height ?? (bb.maxY - bb.minY);
    return { x: x0 + w / 2, y: y0 + h / 2 };
  }
  return null;
}

// y-UP: +y renders upward, so dy>0 is 'up'.
function offsetDirection(cx, cy) {
  const dx = cx - SCRATCH.x, dy = cy - SCRATCH.y;
  if (Math.abs(dx) < 1e-6 && Math.abs(dy) < 1e-6) return null;
  return Math.abs(dx) >= Math.abs(dy)
    ? (dx > 0 ? 'right' : 'left')
    : (dy > 0 ? 'up' : 'down');
}

const report = { families: {}, ok: true, notes: [] };

for (const [family, kind] of Object.entries(FAMILY_KIND)) {
  const measured = {};
  for (const rot of ROTATIONS) {
    let flag;
    try {
      flag = family === 'port'
        ? await eda.sch_PrimitiveComponent.createNetPort(kind, 'CAL', SCRATCH.x, SCRATCH.y, rot)
        : await eda.sch_PrimitiveComponent.createNetFlag(kind, 'CAL', SCRATCH.x, SCRATCH.y, rot);
    } catch (e) {
      report.notes.push(`${family}@${rot}: create failed: ${e.message}`);
      report.ok = false;
      continue;
    }
    const pid = flag.getState_PrimitiveId();
    let dir = null;
    try {
      const bb = await eda.sch_Primitive.getPrimitivesBBox([pid]);
      dir = offsetDirection(bboxCenter(bb)?.x, bboxCenter(bb)?.y);
    } catch (e) {
      report.notes.push(`${family}@${rot}: bbox failed: ${e.message}`);
    }
    try { await eda.sch_PrimitiveComponent.delete([pid]); } catch (e) { /* best effort */ }
    measured[rot] = dir;
  }

  const anchorGot = measured[0];
  const anchorWant = EXPECTED.bodyAnchorAtRot0[family];
  // Cycle observed as rotation increases by 90.
  const cycleGot = ROTATIONS.map(r => measured[r]);
  const anchorIdx = EXPECTED.rotationCycle.indexOf(anchorWant);
  const cycleWant = ROTATIONS.map((_, i) =>
    EXPECTED.rotationCycle[(anchorIdx + i) % 4]);
  const pass = anchorGot === anchorWant &&
    JSON.stringify(cycleGot) === JSON.stringify(cycleWant);
  if (!pass) report.ok = false;
  report.families[family] = { measured, anchorGot, anchorWant, cycleGot, cycleWant, pass };
}

report.summary = report.ok
  ? 'PASS — live anchors match orientation.json'
  : 'FAIL — live anchors DIFFER from orientation.json; update both files + re-freeze goldens';
return report;
