import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { StopCircle, Clock } from "lucide-react";
import { useDownloadProgress } from "@/hooks/useDownloadProgress";
interface DownloadProgressProps {
    progress: number;
    remainingCount?: number;
    currentTrack: {
        name: string;
        artists: string;
    } | null;
    onStop: () => void;
}
export function DownloadProgress({ progress, remainingCount = 0, currentTrack, onStop }: DownloadProgressProps) {
    const liveProgress = useDownloadProgress();
    const isCooldown = Boolean(liveProgress.cooldown) && (liveProgress.cooldown_secs ?? 0) > 0;
    const cooldownSecs = liveProgress.cooldown_secs ?? 0;
    const cooldownMessage = liveProgress.cooldown_message ?? "";
    const isRateLimited = Boolean(liveProgress.rate_limited) && (liveProgress.rate_limit_secs ?? 0) > 0;
    const rateLimitSecs = liveProgress.rate_limit_secs ?? 0;
    const clampedProgress = Math.min(100, Math.max(0, progress));
    const safeRemainingCount = Math.max(0, remainingCount);
    const remainingLabel = `${safeRemainingCount.toLocaleString()} ${safeRemainingCount === 1 ? "track" : "tracks"} left`;
    return (<div className="w-full space-y-2 mt-4">
      <div className="flex items-center gap-2">
        <Progress value={clampedProgress} className="h-2 flex-1"/>
        <Button variant="destructive" size="sm" onClick={onStop} className="gap-1.5">
          <StopCircle className="h-4 w-4"/>
          Stop
        </Button>
      </div>
      {isCooldown ? (<p className="flex items-center gap-1.5 text-xs text-amber-600 dark:text-amber-400">
          <Clock className="h-3.5 w-3.5 shrink-0"/>
          {cooldownMessage || "The server is taking a scheduled short break."} (back in {cooldownSecs}s)
        </p>) : isRateLimited ? (<p className="flex items-center gap-1.5 text-xs text-amber-600 dark:text-amber-400">
          <Clock className="h-3.5 w-3.5 shrink-0"/>
          Rate limited, please wait. Retrying in {rateLimitSecs}s...
        </p>) : (<p className="text-xs text-muted-foreground">
        {clampedProgress}% • {remainingLabel} -{" "}
        {currentTrack
                ? `${currentTrack.name} - ${currentTrack.artists}`
                : "Preparing download..."}
      </p>)}
    </div>);
}
