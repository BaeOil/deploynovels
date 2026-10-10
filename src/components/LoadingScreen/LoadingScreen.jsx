import "./LoadingScreen.css";

const LoadingScreen = ({ message = "กำลังโหลดข้อมูล...", compact = false }) => {
    return (
        <div className={`loading-screen${compact ? " loading-screen--compact" : ""}`} role="status" aria-live="polite">
            <div className="loading-screen__spinner" aria-hidden="true" />
            <h3 className="loading-screen__message">{message}</h3>
        </div>
    );
};

export default LoadingScreen;