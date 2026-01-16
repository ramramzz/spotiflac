import { useEffect, useState, useMemo } from "react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Checkbox } from "@/components/ui/checkbox";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Pagination,
  PaginationContent,
  PaginationEllipsis,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  Search,
  ArrowUpDown,
  Heart,
  ListMusic,
  LogIn,
  LogOut,
  Download,
  Loader2,
  RefreshCw,
  ExternalLink,
  User,
  CheckCircle2,
  XCircle,
  SkipForward,
  Music,
} from "lucide-react";
import { useSpotifyLibrary, type LibraryTab } from "@/hooks/useSpotifyLibrary";
import { useDownload } from "@/hooks/useDownload";
import { getSettings } from "@/lib/settings";
import { toastWithSound as toast } from "@/lib/toast-with-sound";
import { openExternal } from "@/lib/utils";
import type { LibraryTrack, UserPlaylist } from "@/types/spotify-library";
import type { TrackMetadata } from "@/types/api";

const ITEMS_PER_PAGE = 50;

interface SpotifyLibraryPageProps {
  onPlaylistSelect?: (playlistUrl: string) => void;
}

export function SpotifyLibraryPage({ onPlaylistSelect }: SpotifyLibraryPageProps) {
  const library = useSpotifyLibrary();
  const download = useDownload();

  const [searchQuery, setSearchQuery] = useState("");
  const [sortBy, setSortBy] = useState("default");
  const [currentPage, setCurrentPage] = useState(1);

  useEffect(() => {
    if (library.isAuthenticated && library.likedSongs.length === 0) {
      library.loadLikedSongs(true);
    }
    if (library.isAuthenticated && library.playlists.length === 0) {
      library.loadPlaylists(true);
    }
  }, [library.isAuthenticated]);

  useEffect(() => {
    setCurrentPage(1);
    library.clearSelection();
  }, [library.activeTab]);

  const filteredLikedSongs = useMemo(() => {
    let result = [...library.likedSongs];

    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      result = result.filter(
        (track) =>
          track.name.toLowerCase().includes(query) ||
          track.artists.toLowerCase().includes(query) ||
          track.album.toLowerCase().includes(query)
      );
    }

    result.sort((a, b) => {
      switch (sortBy) {
        case "title_asc":
          return a.name.localeCompare(b.name);
        case "title_desc":
          return b.name.localeCompare(a.name);
        case "artist_asc":
          return a.artists.localeCompare(b.artists);
        case "artist_desc":
          return b.artists.localeCompare(a.artists);
        case "added_desc":
          return new Date(b.added_at).getTime() - new Date(a.added_at).getTime();
        case "added_asc":
          return new Date(a.added_at).getTime() - new Date(b.added_at).getTime();
        case "duration_asc":
          return a.duration_ms - b.duration_ms;
        case "duration_desc":
          return b.duration_ms - a.duration_ms;
        default:
          return new Date(b.added_at).getTime() - new Date(a.added_at).getTime();
      }
    });

    return result;
  }, [library.likedSongs, searchQuery, sortBy]);

  const filteredPlaylists = useMemo(() => {
    let result = [...library.playlists];

    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      result = result.filter(
        (playlist) =>
          playlist.name.toLowerCase().includes(query) ||
          playlist.owner_name.toLowerCase().includes(query)
      );
    }

    result.sort((a, b) => {
      switch (sortBy) {
        case "title_asc":
          return a.name.localeCompare(b.name);
        case "title_desc":
          return b.name.localeCompare(a.name);
        case "tracks_desc":
          return b.track_count - a.track_count;
        case "tracks_asc":
          return a.track_count - b.track_count;
        default:
          return 0;
      }
    });

    return result;
  }, [library.playlists, searchQuery, sortBy]);

  const totalPages =
    library.activeTab === "liked"
      ? Math.ceil(filteredLikedSongs.length / ITEMS_PER_PAGE)
      : Math.ceil(filteredPlaylists.length / ITEMS_PER_PAGE);

  const startIndex = (currentPage - 1) * ITEMS_PER_PAGE;

  const paginatedTracks = filteredLikedSongs.slice(
    startIndex,
    startIndex + ITEMS_PER_PAGE
  );

  const paginatedPlaylists = filteredPlaylists.slice(
    startIndex,
    startIndex + ITEMS_PER_PAGE
  );

  const selectableTracks = paginatedTracks.filter((t) => t.isrc);

  const handleSelectAll = () => {
    library.selectAllTracks(selectableTracks);
  };

  const convertToTrackMetadata = (track: LibraryTrack): TrackMetadata => ({
    id: track.id,
    spotify_id: track.spotify_id,
    name: track.name,
    artists: track.artists,
    artist_ids: track.artist_ids,
    album_name: track.album,
    album_id: track.album_id,
    album_artist: track.album_artist,
    duration: track.duration,
    duration_ms: track.duration_ms,
    images: track.cover_url,
    isrc: track.isrc,
    track_number: track.track_number,
    disc_number: track.disc_number,
    total_tracks: track.total_tracks,
    release_date: track.release_date,
    explicit: track.explicit,
    external_urls: `https://open.spotify.com/track/${track.spotify_id}`,
  });

  const handleDownloadTrack = async (track: LibraryTrack) => {
    if (!track.isrc) {
      toast.error("This track doesn't have an ISRC");
      return;
    }

    await download.handleDownloadTrack(
      track.isrc,
      track.name,
      track.artists,
      track.album,
      track.spotify_id,
      "Liked Songs",
      track.duration_ms,
      1,
      track.album_artist,
      track.release_date,
      track.cover_url,
      track.track_number,
      track.disc_number,
      track.total_tracks,
      1
    );
  };

  const handleDownloadSelected = async () => {
    const selectedTrackObjects = library.likedSongs.filter((t) =>
      library.selectedTracks.includes(t.isrc)
    );

    if (selectedTrackObjects.length === 0) {
      toast.error("No tracks selected");
      return;
    }

    const trackMetadataList = selectedTrackObjects.map(convertToTrackMetadata);
    await download.handleDownloadSelected(
      library.selectedTracks,
      trackMetadataList,
      "Liked Songs"
    );
  };

  const handleDownloadAll = async () => {
    const tracksWithIsrc = filteredLikedSongs.filter((t) => t.isrc);
    if (tracksWithIsrc.length === 0) {
      toast.error("No tracks available for download");
      return;
    }

    const trackMetadataList = tracksWithIsrc.map(convertToTrackMetadata);
    await download.handleDownloadAll(trackMetadataList, "Liked Songs");
  };

  const handlePlaylistClick = (playlist: UserPlaylist) => {
    if (onPlaylistSelect) {
      onPlaylistSelect(playlist.spotify_url);
    } else {
      openExternal(playlist.spotify_url);
    }
  };

  const getPaginationPages = (
    current: number,
    total: number
  ): (number | "ellipsis")[] => {
    if (total <= 10) {
      return Array.from({ length: total }, (_, i) => i + 1);
    }
    const pages: (number | "ellipsis")[] = [];
    pages.push(1);
    if (current <= 7) {
      for (let i = 2; i <= 10; i++) pages.push(i);
      pages.push("ellipsis");
      pages.push(total);
    } else if (current >= total - 7) {
      pages.push("ellipsis");
      for (let i = total - 9; i <= total; i++) pages.push(i);
    } else {
      pages.push("ellipsis");
      pages.push(current - 1);
      pages.push(current);
      pages.push(current + 1);
      pages.push("ellipsis");
      pages.push(total);
    }
    return pages;
  };

  if (library.isCheckingAuth) {
    return (
      <div className="flex flex-col items-center justify-center p-16 text-center gap-4">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
        <p className="text-muted-foreground">Checking authentication...</p>
      </div>
    );
  }

  if (!library.isAuthenticated) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-2">
          <h2 className="text-2xl font-bold tracking-tight">Spotify Library</h2>
        </div>

        <div className="flex flex-col items-center justify-center p-16 text-center gap-6 border rounded-lg bg-card">
          <div className="rounded-full bg-primary/10 p-6">
            <Music className="h-12 w-12 text-primary" />
          </div>
          <div className="space-y-2 max-w-md">
            <h3 className="text-xl font-semibold">Connect Your Spotify Account</h3>
            <p className="text-muted-foreground">
              Sign in with your Spotify account to access your liked songs and
              playlists for downloading.
            </p>
          </div>
          <Button
            size="lg"
            onClick={library.startAuth}
            disabled={library.isAuthenticating}
            className="gap-2"
          >
            {library.isAuthenticating ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin" />
                Connecting...
              </>
            ) : (
              <>
                <LogIn className="h-4 w-4" />
                Connect with Spotify
              </>
            )}
          </Button>
          <p className="text-xs text-muted-foreground max-w-sm">
            Your browser will open to authorize SpotiFLAC. After authorization,
            you'll be redirected back automatically.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <h2 className="text-2xl font-bold tracking-tight">Spotify Library</h2>
            {library.user && (
              <div className="flex items-center gap-2 px-3 py-1.5 rounded-full bg-muted/50">
                {library.user.image_url ? (
                  <img
                    src={library.user.image_url}
                    alt={library.user.display_name}
                    className="h-5 w-5 rounded-full"
                  />
                ) : (
                  <User className="h-4 w-4 text-muted-foreground" />
                )}
                <span className="text-sm font-medium">
                  {library.user.display_name}
                </span>
              </div>
            )}
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={library.logout}
            className="gap-2"
          >
            <LogOut className="h-4 w-4" />
            Disconnect
          </Button>
        </div>

        <Tabs
          value={library.activeTab}
          onValueChange={(v) => library.setActiveTab(v as LibraryTab)}
        >
          <TabsList className="grid w-full grid-cols-2 max-w-md">
            <TabsTrigger value="liked" className="gap-2">
              <Heart className="h-4 w-4" />
              Liked Songs
              {library.likedSongsTotal > 0 && (
                <Badge variant="secondary" className="ml-1 font-mono text-xs">
                  {library.likedSongsTotal.toLocaleString()}
                </Badge>
              )}
            </TabsTrigger>
            <TabsTrigger value="playlists" className="gap-2">
              <ListMusic className="h-4 w-4" />
              Playlists
              {library.playlistsTotal > 0 && (
                <Badge variant="secondary" className="ml-1 font-mono text-xs">
                  {library.playlistsTotal.toLocaleString()}
                </Badge>
              )}
            </TabsTrigger>
          </TabsList>
        </Tabs>

        <div className="flex items-center gap-2">
          <div className="relative flex-1">
            <Search className="absolute left-2 top-2.5 h-4 w-4 text-muted-foreground" />
            <Input
              placeholder={
                library.activeTab === "liked"
                  ? "Search liked songs..."
                  : "Search playlists..."
              }
              value={searchQuery}
              onChange={(e) => {
                setSearchQuery(e.target.value);
                setCurrentPage(1);
              }}
              className="pl-8 h-9"
            />
          </div>
          <Select value={sortBy} onValueChange={setSortBy}>
            <SelectTrigger className="w-[180px] h-9">
              <ArrowUpDown className="mr-2 h-4 w-4" />
              <SelectValue placeholder="Sort by" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="default">Default</SelectItem>
              {library.activeTab === "liked" ? (
                <>
                  <SelectItem value="added_desc">Recently Added</SelectItem>
                  <SelectItem value="added_asc">Oldest First</SelectItem>
                  <SelectItem value="title_asc">Title (A-Z)</SelectItem>
                  <SelectItem value="title_desc">Title (Z-A)</SelectItem>
                  <SelectItem value="artist_asc">Artist (A-Z)</SelectItem>
                  <SelectItem value="artist_desc">Artist (Z-A)</SelectItem>
                  <SelectItem value="duration_asc">Duration (Short)</SelectItem>
                  <SelectItem value="duration_desc">Duration (Long)</SelectItem>
                </>
              ) : (
                <>
                  <SelectItem value="title_asc">Name (A-Z)</SelectItem>
                  <SelectItem value="title_desc">Name (Z-A)</SelectItem>
                  <SelectItem value="tracks_desc">Most Tracks</SelectItem>
                  <SelectItem value="tracks_asc">Fewest Tracks</SelectItem>
                </>
              )}
            </SelectContent>
          </Select>
          <TooltipProvider>
            <Tooltip delayDuration={0}>
              <TooltipTrigger asChild>
                <Button
                  variant="outline"
                  size="icon"
                  className="h-9 w-9"
                  onClick={() =>
                    library.activeTab === "liked"
                      ? library.loadLikedSongs(true)
                      : library.loadPlaylists(true)
                  }
                  disabled={
                    library.isLoadingLikedSongs || library.isLoadingPlaylists
                  }
                >
                  <RefreshCw
                    className={`h-4 w-4 ${
                      library.isLoadingLikedSongs || library.isLoadingPlaylists
                        ? "animate-spin"
                        : ""
                    }`}
                  />
                </Button>
              </TooltipTrigger>
              <TooltipContent>Refresh</TooltipContent>
            </Tooltip>
          </TooltipProvider>
        </div>

        {library.activeTab === "liked" && library.selectedTracks.length > 0 && (
          <div className="flex items-center gap-2 p-3 bg-muted/50 rounded-lg">
            <Badge variant="secondary">
              {library.selectedTracks.length} selected
            </Badge>
            <div className="flex-1" />
            <Button
              size="sm"
              variant="outline"
              onClick={() => library.clearSelection()}
            >
              Clear Selection
            </Button>
            <Button
              size="sm"
              onClick={handleDownloadSelected}
              disabled={download.isDownloading}
              className="gap-2"
            >
              {download.isDownloading ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Download className="h-4 w-4" />
              )}
              Download Selected
            </Button>
          </div>
        )}
      </div>

      {library.activeTab === "liked" ? (
        <div className="rounded-md border overflow-hidden">
          {library.isLoadingLikedSongs && library.likedSongs.length === 0 ? (
            <div className="flex flex-col items-center justify-center p-16 text-center gap-3">
              <Loader2 className="h-8 w-8 animate-spin text-primary" />
              <p className="text-muted-foreground">Loading liked songs...</p>
            </div>
          ) : filteredLikedSongs.length === 0 ? (
            <div className="flex flex-col items-center justify-center p-16 text-center text-muted-foreground gap-3">
              <div className="rounded-full bg-muted/50 p-4 ring-8 ring-muted/20">
                <Heart className="h-10 w-10 opacity-40" />
              </div>
              <div className="space-y-1">
                <p className="font-medium text-foreground/80">No liked songs</p>
                <p className="text-sm">
                  {searchQuery
                    ? "No tracks match your search."
                    : "Like some songs on Spotify to see them here."}
                </p>
              </div>
            </div>
          ) : (
            <>
              <table className="w-full table-fixed">
                <thead>
                  <tr className="border-b bg-muted/50">
                    <th className="h-10 px-4 text-left align-middle font-medium text-muted-foreground w-12">
                      <Checkbox
                        checked={
                          selectableTracks.length > 0 &&
                          selectableTracks.every((t) =>
                            library.selectedTracks.includes(t.isrc)
                          )
                        }
                        onCheckedChange={handleSelectAll}
                      />
                    </th>
                    <th className="h-10 px-4 text-left align-middle font-medium text-muted-foreground w-12 text-xs uppercase">
                      #
                    </th>
                    <th className="h-10 px-4 text-left align-middle font-medium text-muted-foreground text-xs uppercase">
                      Title
                    </th>
                    <th className="h-10 px-4 text-left align-middle font-medium text-muted-foreground hidden md:table-cell text-xs uppercase w-1/4">
                      Album
                    </th>
                    <th className="h-10 px-4 text-left align-middle font-medium text-muted-foreground hidden lg:table-cell w-16 text-xs uppercase">
                      Dur
                    </th>
                    <th className="h-10 px-4 text-center align-middle font-medium text-muted-foreground w-24 text-xs uppercase">
                      Actions
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {paginatedTracks.map((track, index) => {
                    const isDownloading =
                      download.downloadingTrack === track.isrc;
                    const isDownloaded = download.downloadedTracks.has(track.isrc);
                    const isFailed = download.failedTracks.has(track.isrc);
                    const isSkipped = download.skippedTracks.has(track.isrc);

                    return (
                      <tr
                        key={track.id}
                        className="border-b transition-colors hover:bg-muted/50"
                      >
                        <td className="p-3 align-middle">
                          <Checkbox
                            checked={library.selectedTracks.includes(track.isrc)}
                            onCheckedChange={() =>
                              library.toggleTrackSelection(track.isrc)
                            }
                            disabled={!track.isrc}
                          />
                        </td>
                        <td className="p-3 align-middle text-sm text-muted-foreground text-left font-mono">
                          {startIndex + index + 1}
                        </td>
                        <td className="p-3 align-middle min-w-0">
                          <div className="flex items-center gap-3 min-w-0">
                            <img
                              src={
                                track.cover_url ||
                                "https://placehold.co/300?text=No+Cover"
                              }
                              alt={track.album}
                              className="h-10 w-10 rounded shrink-0 bg-secondary object-cover"
                              onError={(e) => {
                                (e.target as HTMLImageElement).src =
                                  "https://placehold.co/300?text=No+Cover";
                              }}
                            />
                            <div className="flex flex-col min-w-0 flex-1">
                              <div className="flex items-center gap-2">
                                <span className="font-medium text-sm truncate">
                                  {track.name}
                                </span>
                                {track.explicit && (
                                  <Badge
                                    variant="outline"
                                    className="text-[10px] px-1 py-0"
                                  >
                                    E
                                  </Badge>
                                )}
                              </div>
                              <span className="text-xs text-muted-foreground truncate">
                                {track.artists}
                              </span>
                            </div>
                          </div>
                        </td>
                        <td className="p-3 align-middle text-sm text-muted-foreground hidden md:table-cell">
                          <div className="truncate">{track.album}</div>
                        </td>
                        <td className="p-3 align-middle text-sm text-muted-foreground hidden lg:table-cell font-mono">
                          {track.duration}
                        </td>
                        <td className="p-3 align-middle">
                          <div className="flex items-center justify-center gap-1">
                            {isDownloaded ? (
                              <TooltipProvider>
                                <Tooltip delayDuration={0}>
                                  <TooltipTrigger asChild>
                                    <div className="h-8 w-8 flex items-center justify-center">
                                      {isSkipped ? (
                                        <SkipForward className="h-4 w-4 text-muted-foreground" />
                                      ) : (
                                        <CheckCircle2 className="h-4 w-4 text-green-500" />
                                      )}
                                    </div>
                                  </TooltipTrigger>
                                  <TooltipContent>
                                    {isSkipped ? "Skipped (exists)" : "Downloaded"}
                                  </TooltipContent>
                                </Tooltip>
                              </TooltipProvider>
                            ) : isFailed ? (
                              <TooltipProvider>
                                <Tooltip delayDuration={0}>
                                  <TooltipTrigger asChild>
                                    <div className="h-8 w-8 flex items-center justify-center">
                                      <XCircle className="h-4 w-4 text-destructive" />
                                    </div>
                                  </TooltipTrigger>
                                  <TooltipContent>Download failed</TooltipContent>
                                </Tooltip>
                              </TooltipProvider>
                            ) : (
                              <TooltipProvider>
                                <Tooltip delayDuration={0}>
                                  <TooltipTrigger asChild>
                                    <Button
                                      variant="ghost"
                                      size="icon"
                                      className="h-8 w-8"
                                      onClick={() => handleDownloadTrack(track)}
                                      disabled={
                                        isDownloading ||
                                        !track.isrc ||
                                        download.isDownloading
                                      }
                                    >
                                      {isDownloading ? (
                                        <Loader2 className="h-4 w-4 animate-spin" />
                                      ) : (
                                        <Download className="h-4 w-4" />
                                      )}
                                    </Button>
                                  </TooltipTrigger>
                                  <TooltipContent>
                                    {track.isrc ? "Download" : "No ISRC available"}
                                  </TooltipContent>
                                </Tooltip>
                              </TooltipProvider>
                            )}
                            <TooltipProvider>
                              <Tooltip delayDuration={0}>
                                <TooltipTrigger asChild>
                                  <Button
                                    variant="ghost"
                                    size="icon"
                                    className="h-8 w-8"
                                    onClick={() =>
                                      openExternal(
                                        `https://open.spotify.com/track/${track.spotify_id}`
                                      )
                                    }
                                  >
                                    <ExternalLink className="h-4 w-4" />
                                  </Button>
                                </TooltipTrigger>
                                <TooltipContent>Open in Spotify</TooltipContent>
                              </Tooltip>
                            </TooltipProvider>
                          </div>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>

              {filteredLikedSongs.length > 0 && (
                <div className="flex items-center justify-between p-3 bg-muted/30 border-t">
                  <Button
                    size="sm"
                    onClick={handleDownloadAll}
                    disabled={download.isDownloading}
                    className="gap-2"
                  >
                    {download.isDownloading ? (
                      <Loader2 className="h-4 w-4 animate-spin" />
                    ) : (
                      <Download className="h-4 w-4" />
                    )}
                    Download All ({filteredLikedSongs.filter((t) => t.isrc).length})
                  </Button>
                  {download.isDownloading && (
                    <Button
                      size="sm"
                      variant="destructive"
                      onClick={download.handleStopDownload}
                    >
                      Stop Download
                    </Button>
                  )}
                </div>
              )}
            </>
          )}
        </div>
      ) : (
        <div className="rounded-md border overflow-hidden">
          {library.isLoadingPlaylists && library.playlists.length === 0 ? (
            <div className="flex flex-col items-center justify-center p-16 text-center gap-3">
              <Loader2 className="h-8 w-8 animate-spin text-primary" />
              <p className="text-muted-foreground">Loading playlists...</p>
            </div>
          ) : filteredPlaylists.length === 0 ? (
            <div className="flex flex-col items-center justify-center p-16 text-center text-muted-foreground gap-3">
              <div className="rounded-full bg-muted/50 p-4 ring-8 ring-muted/20">
                <ListMusic className="h-10 w-10 opacity-40" />
              </div>
              <div className="space-y-1">
                <p className="font-medium text-foreground/80">No playlists</p>
                <p className="text-sm">
                  {searchQuery
                    ? "No playlists match your search."
                    : "Create or follow playlists on Spotify to see them here."}
                </p>
              </div>
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-3 p-4">
              {paginatedPlaylists.map((playlist) => (
                <div
                  key={playlist.id}
                  className="flex items-center gap-3 p-3 rounded-lg border bg-card hover:bg-muted/50 transition-colors cursor-pointer group"
                  onClick={() => handlePlaylistClick(playlist)}
                >
                  <img
                    src={playlist.cover_url || "https://placehold.co/300?text=No+Cover"}
                    alt={playlist.name}
                    className="h-16 w-16 rounded shrink-0 bg-secondary object-cover"
                    onError={(e) => {
                      (e.target as HTMLImageElement).src =
                        "https://placehold.co/300?text=No+Cover";
                    }}
                  />
                  <div className="flex flex-col min-w-0 flex-1">
                    <span className="font-medium text-sm truncate group-hover:text-primary transition-colors">
                      {playlist.name}
                    </span>
                    <span className="text-xs text-muted-foreground truncate">
                      by {playlist.owner_name}
                    </span>
                    <span className="text-xs text-muted-foreground mt-1">
                      {playlist.track_count.toLocaleString()} tracks
                    </span>
                  </div>
                  <ExternalLink className="h-4 w-4 text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity shrink-0" />
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {totalPages > 1 && (
        <Pagination>
          <PaginationContent>
            <PaginationItem>
              <PaginationPrevious
                href="#"
                onClick={(e) => {
                  e.preventDefault();
                  if (currentPage > 1) setCurrentPage(currentPage - 1);
                }}
                className={
                  currentPage === 1 ? "pointer-events-none opacity-50" : "cursor-pointer"
                }
              />
            </PaginationItem>

            {getPaginationPages(currentPage, totalPages).map((page, index) =>
              page === "ellipsis" ? (
                <PaginationItem key={`ellipsis-${index}`}>
                  <PaginationEllipsis />
                </PaginationItem>
              ) : (
                <PaginationItem key={page}>
                  <PaginationLink
                    href="#"
                    onClick={(e) => {
                      e.preventDefault();
                      setCurrentPage(page);
                    }}
                    isActive={currentPage === page}
                    className="cursor-pointer"
                  >
                    {page}
                  </PaginationLink>
                </PaginationItem>
              )
            )}

            <PaginationItem>
              <PaginationNext
                href="#"
                onClick={(e) => {
                  e.preventDefault();
                  if (currentPage < totalPages) setCurrentPage(currentPage + 1);
                }}
                className={
                  currentPage === totalPages
                    ? "pointer-events-none opacity-50"
                    : "cursor-pointer"
                }
              />
            </PaginationItem>
          </PaginationContent>
        </Pagination>
      )}
    </div>
  );
}
