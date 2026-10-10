export const createImage = (url) =>
  new Promise((resolve, reject) => {
    const image = new Image();
    image.addEventListener("load", () => resolve(image));
    image.addEventListener("error", (error) => reject(error));
    image.setAttribute("crossOrigin", "anonymous"); // ป้องกันปัญหา CORS
    image.src = url;
  });

export default async function getCroppedImg(imageSrc, pixelCrop) {
  const image = await createImage(imageSrc);
  const canvas = document.createElement("canvas");
  const ctx = canvas.getContext("2d");

  if (!ctx) return null;

  // ตั้งค่าขนาด Canvas ตามพื้นที่ที่ผู้ใช้ครอบรูป
  canvas.width = pixelCrop.width;
  canvas.height = pixelCrop.height;

  // วาดรูปที่ถูกตัดลงบน Canvas
  ctx.drawImage(
    image,
    pixelCrop.x,
    pixelCrop.y,
    pixelCrop.width,
    pixelCrop.height,
    0,
    0,
    pixelCrop.width,
    pixelCrop.height
  );

  // แปลง Canvas ให้กลายเป็นไฟล์ Blob เพื่อส่งไปใช้งานต่อ
  return new Promise((resolve, reject) => {
    canvas.toBlob((file) => {
      if (!file) {
        reject(new Error("Canvas is empty"));
        return;
      }
      file.name = "cropped-cover.jpeg";
      resolve({
        file: file,
        url: URL.createObjectURL(file)
      });
    }, "image/jpeg", 0.95); // เซฟเป็น jpeg คุณภาพ 95%
  });
}