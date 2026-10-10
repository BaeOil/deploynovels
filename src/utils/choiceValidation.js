const normalizeId = (value) => value === undefined || value === null ? "" : String(value);

const getChoiceSourceId = (choice) => choice?.from_scene_id ?? choice?.fromSceneID ?? choice?.FromSceneID ?? choice?.from;
const getChoiceTargetId = (choice) => choice?.to_scene_id ?? choice?.toSceneID ?? choice?.ToSceneID ?? choice?.target_scene_id ?? choice?.to;

const getSceneFromChapters = (sceneId, chapters) => {
  const normalizedSceneId = normalizeId(sceneId);
  return (chapters || []).flatMap((chapter) => (chapter?.scenes ?? chapter?.Scenes) || [])
    .find((scene) => normalizeId(scene?.id ?? scene?.ID ?? scene?.scene_id ?? scene?.SceneID) === normalizedSceneId);
};

export const getChoiceConnectionBlockReason = (sourceId, targetId, chapters, currentChoiceId) => {
  const source = normalizeId(sourceId);
  const target = normalizeId(targetId);
  if (!source || !target) return "invalid";
  if (source === target) return "self";

  const sourceScene = getSceneFromChapters(source, chapters);
  const targetScene = getSceneFromChapters(target, chapters);
  const sourceType = String(sourceScene?.type ?? sourceScene?.Type ?? "").toLowerCase();
  const targetType = String(targetScene?.type ?? targetScene?.Type ?? "").toLowerCase();

  if (sourceType === "ending" || sourceScene?.is_ending === true || sourceScene?.IsEnding === true || String(sourceScene?.status ?? sourceScene?.Status ?? "").toLowerCase() === "ending") {
    return "source_ending";
  }
  if (targetType === "start" || targetType === "starting" || String(targetScene?.status ?? targetScene?.Status ?? "").toLowerCase() === "start" || targetScene?.is_start_scene === true || targetScene?.isStart === true) {
    return "target_start";
  }

  const reverseAdjacency = new Map();
  (chapters || []).forEach((chapter) => {
    ((chapter?.scenes ?? chapter?.Scenes) || []).forEach((scene) => {
      ((scene?.choices ?? scene?.Choices) || []).forEach((choice) => {
        const choiceId = normalizeId(choice?.id ?? choice?.ID ?? choice?.choice_id ?? choice?.ChoiceID);
        if (currentChoiceId && choiceId === normalizeId(currentChoiceId)) return;

        const from = normalizeId(getChoiceSourceId(choice));
        const to = normalizeId(getChoiceTargetId(choice));
        if (!from || !to) return;
        if (!reverseAdjacency.has(to)) reverseAdjacency.set(to, []);
        reverseAdjacency.get(to).push(from);
      });
    });
  });

  const visited = new Set();
  const queue = [source];
  while (queue.length > 0) {
    const current = queue.shift();
    if (current === target) return "cycle";
    if (visited.has(current)) continue;
    visited.add(current);
    (reverseAdjacency.get(current) || []).forEach((parentId) => queue.push(parentId));
  }
  return null;
};

export const getChoiceConnectionMessage = (reason) => ({
  invalid: "กรุณาเลือกฉากปลายทางที่ถูกต้อง",
  self: "ไม่สามารถเชื่อมฉากเข้าหาตัวเองได้",
  cycle: "ไม่สามารถเชื่อมได้ เพราะจะเกิดลูปในโครงสร้างเรื่อง",
  source_ending: "ฉากจบไม่สามารถสร้างทางเลือกออกไปได้",
  target_start: "ไม่สามารถเชื่อมไปยังฉากเริ่มต้นได้",
  backward: "ไม่สามารถเชื่อมย้อนกลับไปยังฉากก่อนหน้าได้",
}[reason] || "เส้นทางนี้ไม่ถูกต้องตามกฎโครงสร้างเรื่อง");
