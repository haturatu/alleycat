import React from "react";
import ReactDOM from "react-dom/client";
import App from "./App";

const loadScript = (src: string, attrs: Record<string, string> = {}) => {
  const script = document.createElement("script");
  script.src = src;
  script.async = true;
  Object.entries(attrs).forEach(([key, value]) => {
    script.setAttribute(key, value);
  });
  document.head.appendChild(script);
};

const isAdminApp = import.meta.env.VITE_ADMIN === "true";
if (!isAdminApp) {
  const analyticsUrl = import.meta.env.VITE_ANALYTICS_URL;
  const analyticsSiteId = import.meta.env.VITE_ANALYTICS_SITE_ID;
  const adsClient = import.meta.env.VITE_ADS_CLIENT;

  window.setTimeout(() => {
    if (analyticsUrl && analyticsSiteId) {
      loadScript(analyticsUrl, { "data-website-id": analyticsSiteId });
    }
    if (adsClient) {
      loadScript(`https://pagead2.googlesyndication.com/pagead/js/adsbygoogle.js?client=${adsClient}`, {
        crossorigin: "anonymous",
      });
    }
  }, 0);
}

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
