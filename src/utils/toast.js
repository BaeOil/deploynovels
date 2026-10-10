// Minimal toast utility — no React required, appends to document.body
const TOAST_CONTAINER_ID = "app-toast-container";

function ensureContainer() {
  let c = document.getElementById(TOAST_CONTAINER_ID);
  if (c) return c;
  c = document.createElement("div");
  c.id = TOAST_CONTAINER_ID;
  Object.assign(c.style, {
    position: "fixed",
    right: "16px",
    bottom: "16px",
    zIndex: 9999,
    display: "flex",
    flexDirection: "column",
    gap: "8px",
    alignItems: "flex-end",
    pointerEvents: "none",
  });
  document.body.appendChild(c);
  return c;
}

export function showToast(message, { duration = 3500, type = "info" } = {}) {
  if (typeof document === "undefined") return;
  const container = ensureContainer();

  const el = document.createElement("div");
  el.className = `app-toast app-toast--${type}`;
  el.textContent = message;
  Object.assign(el.style, {
    background: type === "error" ? "#F87171" : type === "success" ? "#34D399" : "#111827",
    color: "#fff",
    padding: "10px 14px",
    borderRadius: "10px",
    boxShadow: "0 6px 20px rgba(2,6,23,0.12)",
    fontSize: "14px",
    pointerEvents: "auto",
    opacity: "0",
    transform: "translateY(8px)",
    transition: "opacity 220ms ease, transform 220ms ease",
    maxWidth: "320px",
  });

  container.appendChild(el);

  // animate in
  requestAnimationFrame(() => {
    el.style.opacity = "1";
    el.style.transform = "translateY(0)";
  });

  const timeout = setTimeout(() => {
    hide();
  }, duration);

  function hide() {
    clearTimeout(timeout);
    el.style.opacity = "0";
    el.style.transform = "translateY(8px)";
    setTimeout(() => {
      el.remove();
    }, 240);
  }

  el.addEventListener("click", hide);
  return { hide };
}

export default { showToast };
