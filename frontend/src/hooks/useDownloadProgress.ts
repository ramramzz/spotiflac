import { useState, useEffect, useRef } from "react";
import { GetDownloadProgress } from "../../wailsjs/go/main/App";
export interface DownloadProgressInfo {
    is_downloading: boolean;
    mb_downloaded: number;
    speed_mbps: number;
    rate_limited?: boolean;
    rate_limit_secs?: number;
    cooldown?: boolean;
    cooldown_secs?: number;
    cooldown_message?: string;
    cooldown_event_id?: number;
}
export function useDownloadProgress() {
    const [progress, setProgress] = useState<DownloadProgressInfo>({
        is_downloading: false,
        mb_downloaded: 0,
        speed_mbps: 0,
        rate_limited: false,
        rate_limit_secs: 0,
        cooldown: false,
        cooldown_secs: 0,
        cooldown_message: "",
    });
    const intervalRef = useRef<number | null>(null);
    useEffect(() => {
        const pollProgress = async () => {
            try {
                const progressInfo = await GetDownloadProgress();
                setProgress(progressInfo);
            }
            catch (error) {
                console.error("Failed to get download progress:", error);
            }
        };
        intervalRef.current = window.setInterval(pollProgress, 200);
        pollProgress();
        return () => {
            if (intervalRef.current) {
                clearInterval(intervalRef.current);
            }
        };
    }, []);
    return progress;
}
