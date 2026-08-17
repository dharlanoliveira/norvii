import { BrowserRouter } from "react-router-dom";

import "../i18n/config";
import { AppRoutes } from "./routes";

export function App() {
  return (
    <BrowserRouter>
      <AppRoutes />
    </BrowserRouter>
  );
}
