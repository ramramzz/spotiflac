import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { MotionConfig } from "motion/react";
import "./index.css";
import App from "./App.tsx";
import { Toaster } from "@/components/ui/sonner";
createRoot(document.getElementById("root")!).render(<StrictMode>
    <MotionConfig reducedMotion="user">
      <App />
      <Toaster position="bottom-left" duration={1000}/>
    </MotionConfig>
  </StrictMode>);
