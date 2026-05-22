import React from "react";
import { createRoot } from "react-dom/client";
import App from "./App";
import "@unocss/reset/tailwind.css";
import "virtual:uno.css";
import "./style.css";
import "./i18n/i18n";

// 清除本地存储的库页面状态
localStorage.removeItem('libraryViewMode');
localStorage.removeItem('libraryTagsFilter');
localStorage.removeItem('library_sourceFilter');
localStorage.removeItem('searchQuery');
localStorage.removeItem('categoryListFilter');

const container = document.getElementById("root");

const root = createRoot(container!);

root.render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);
