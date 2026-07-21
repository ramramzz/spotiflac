import { useState } from "react";
import { Clock, X } from "lucide-react";
import { useDownloadProgress } from "@/hooks/useDownloadProgress";
const DISMISSED_KEY = "spotiflac_cooldown_dismissed_event";
export function CooldownBanner() {
    const progress = useDownloadProgress();
    const isCooldown = Boolean(progress.cooldown) && (progress.cooldown_secs ?? 0) > 0;
    const eventID = progress.cooldown_event_id ?? 0;
    const [dismissedEventID, setDismissedEventID] = useState(() => {
        const stored = Number(localStorage.getItem(DISMISSED_KEY));
        return Number.isFinite(stored) ? stored : 0;
    });
    const dismiss = (id: number) => {
        setDismissedEventID(id);
        localStorage.setItem(DISMISSED_KEY, String(id));
    };
    if (!isCooldown || dismissedEventID === eventID) {
        return null;
    }
    const cooldownMessage = progress.cooldown_message?.trim() || "The server is taking a scheduled short break.";
    return (<div className="fixed top-12 left-1/2 z-50 -translate-x-1/2 animate-in fade-in slide-in-from-top-2">
      <div className="flex items-center gap-2.5 rounded-lg border border-amber-300 bg-amber-50 px-3.5 py-2 text-amber-700 shadow-lg dark:border-amber-800 dark:bg-amber-950 dark:text-amber-300">
        <Clock className="h-4 w-4 shrink-0"/>
        <p className="text-xs font-medium leading-tight">
          {cooldownMessage}
        </p>
        <button type="button" onClick={() => dismiss(eventID)} aria-label="Dismiss" className="ml-1 shrink-0 rounded-md p-1 text-amber-600/70 transition-colors hover:bg-amber-100 hover:text-amber-700 dark:text-amber-400/70 dark:hover:bg-amber-900 dark:hover:text-amber-300">
          <X className="h-3.5 w-3.5"/>
        </button>
      </div>
    </div>);
}
