import { useState, useCallback, useEffect } from "react";
import type {
  SpotifyAuthStatus,
  SpotifyUserProfile,
  LibraryTrack,
  LikedSongsResponse,
  UserPlaylist,
  UserPlaylistsResponse,
} from "@/types/spotify-library";
import { toastWithSound as toast } from "@/lib/toast-with-sound";
import { BrowserOpenURL } from "../../wailsjs/runtime/runtime";

const GetSpotifyAuthStatus = (): Promise<SpotifyAuthStatus> =>
  (window as any)["go"]["main"]["App"]["GetSpotifyAuthStatus"]();

const GetSpotifyAuthURL = (): Promise<string> =>
  (window as any)["go"]["main"]["App"]["GetSpotifyAuthURL"]();

const StartSpotifyAuth = (): Promise<string> =>
  (window as any)["go"]["main"]["App"]["StartSpotifyAuth"]();

const LogoutSpotify = (): Promise<void> =>
  (window as any)["go"]["main"]["App"]["LogoutSpotify"]();

const GetSpotifyUserProfile = (): Promise<SpotifyUserProfile> =>
  (window as any)["go"]["main"]["App"]["GetSpotifyUserProfile"]();

const GetSpotifyLikedSongs = (
  limit: number,
  offset: number
): Promise<LikedSongsResponse> =>
  (window as any)["go"]["main"]["App"]["GetSpotifyLikedSongs"](limit, offset);

const GetAllSpotifyLikedSongs = (): Promise<LikedSongsResponse> =>
  (window as any)["go"]["main"]["App"]["GetAllSpotifyLikedSongs"]();

const GetSpotifyUserPlaylists = (
  limit: number,
  offset: number
): Promise<UserPlaylistsResponse> =>
  (window as any)["go"]["main"]["App"]["GetSpotifyUserPlaylists"](limit, offset);

const GetAllSpotifyUserPlaylists = (): Promise<UserPlaylistsResponse> =>
  (window as any)["go"]["main"]["App"]["GetAllSpotifyUserPlaylists"]();

export type LibraryTab = "liked" | "playlists";

export function useSpotifyLibrary() {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [isAuthenticating, setIsAuthenticating] = useState(false);
  const [isCheckingAuth, setIsCheckingAuth] = useState(true);
  const [user, setUser] = useState<SpotifyUserProfile | null>(null);

  const [likedSongs, setLikedSongs] = useState<LibraryTrack[]>([]);
  const [likedSongsTotal, setLikedSongsTotal] = useState(0);
  const [isLoadingLikedSongs, setIsLoadingLikedSongs] = useState(false);

  const [playlists, setPlaylists] = useState<UserPlaylist[]>([]);
  const [playlistsTotal, setPlaylistsTotal] = useState(0);
  const [isLoadingPlaylists, setIsLoadingPlaylists] = useState(false);

  const [activeTab, setActiveTab] = useState<LibraryTab>("liked");
  const [selectedTracks, setSelectedTracks] = useState<string[]>([]);

  const checkAuthStatus = useCallback(async () => {
    setIsCheckingAuth(true);
    try {
      const status = await GetSpotifyAuthStatus();
      setIsAuthenticated(status.is_authenticated);
      if (status.user) {
        setUser({
          id: status.user.id || "",
          display_name: status.user.display_name || "",
          email: status.user.email || "",
          image_url: status.user.image_url || "",
          country: status.user.country || "",
          product: status.user.product || "",
        });
      }
    } catch (err) {
      console.error("Failed to check auth status:", err);
      setIsAuthenticated(false);
    } finally {
      setIsCheckingAuth(false);
    }
  }, []);

  useEffect(() => {
    checkAuthStatus();
  }, [checkAuthStatus]);

  const startAuth = useCallback(async () => {
    setIsAuthenticating(true);
    try {
      const authURL = await GetSpotifyAuthURL();
      BrowserOpenURL(authURL);

      const displayName = await StartSpotifyAuth();
      setIsAuthenticated(true);
      toast.success(`Logged in as ${displayName}`);

      const profile = await GetSpotifyUserProfile();
      setUser(profile);
    } catch (err) {
      console.error("Authentication failed:", err);
      toast.error(
        `Authentication failed: ${err instanceof Error ? err.message : String(err)}`
      );
    } finally {
      setIsAuthenticating(false);
    }
  }, []);

  const logout = useCallback(async () => {
    try {
      await LogoutSpotify();
      setIsAuthenticated(false);
      setUser(null);
      setLikedSongs([]);
      setLikedSongsTotal(0);
      setPlaylists([]);
      setPlaylistsTotal(0);
      setSelectedTracks([]);
      toast.success("Logged out from Spotify");
    } catch (err) {
      console.error("Logout failed:", err);
      toast.error("Failed to logout");
    }
  }, []);

  const loadLikedSongs = useCallback(async (loadAll = false) => {
    setIsLoadingLikedSongs(true);
    try {
      let response: LikedSongsResponse;
      if (loadAll) {
        response = await GetAllSpotifyLikedSongs();
      } else {
        response = await GetSpotifyLikedSongs(50, 0);
      }
      setLikedSongs(response.tracks);
      setLikedSongsTotal(response.total);
    } catch (err) {
      console.error("Failed to load liked songs:", err);
      toast.error("Failed to load liked songs");
    } finally {
      setIsLoadingLikedSongs(false);
    }
  }, []);

  const loadMoreLikedSongs = useCallback(async () => {
    if (likedSongs.length >= likedSongsTotal) return;

    setIsLoadingLikedSongs(true);
    try {
      const response = await GetSpotifyLikedSongs(50, likedSongs.length);
      setLikedSongs((prev) => [...prev, ...response.tracks]);
    } catch (err) {
      console.error("Failed to load more liked songs:", err);
      toast.error("Failed to load more tracks");
    } finally {
      setIsLoadingLikedSongs(false);
    }
  }, [likedSongs.length, likedSongsTotal]);

  const loadPlaylists = useCallback(async (loadAll = false) => {
    setIsLoadingPlaylists(true);
    try {
      let response: UserPlaylistsResponse;
      if (loadAll) {
        response = await GetAllSpotifyUserPlaylists();
      } else {
        response = await GetSpotifyUserPlaylists(50, 0);
      }
      setPlaylists(response.playlists);
      setPlaylistsTotal(response.total);
    } catch (err) {
      console.error("Failed to load playlists:", err);
      toast.error("Failed to load playlists");
    } finally {
      setIsLoadingPlaylists(false);
    }
  }, []);

  const loadMorePlaylists = useCallback(async () => {
    if (playlists.length >= playlistsTotal) return;

    setIsLoadingPlaylists(true);
    try {
      const response = await GetSpotifyUserPlaylists(50, playlists.length);
      setPlaylists((prev) => [...prev, ...response.playlists]);
    } catch (err) {
      console.error("Failed to load more playlists:", err);
      toast.error("Failed to load more playlists");
    } finally {
      setIsLoadingPlaylists(false);
    }
  }, [playlists.length, playlistsTotal]);

  const toggleTrackSelection = useCallback((trackId: string) => {
    setSelectedTracks((prev) =>
      prev.includes(trackId)
        ? prev.filter((id) => id !== trackId)
        : [...prev, trackId]
    );
  }, []);

  const selectAllTracks = useCallback(
    (tracks: LibraryTrack[]) => {
      const trackIds = tracks.filter((t) => t.isrc).map((t) => t.isrc);
      if (selectedTracks.length === trackIds.length) {
        setSelectedTracks([]);
      } else {
        setSelectedTracks(trackIds);
      }
    },
    [selectedTracks.length]
  );

  const clearSelection = useCallback(() => {
    setSelectedTracks([]);
  }, []);

  return {
    isAuthenticated,
    isAuthenticating,
    isCheckingAuth,
    user,
    startAuth,
    logout,

    likedSongs,
    likedSongsTotal,
    isLoadingLikedSongs,
    loadLikedSongs,
    loadMoreLikedSongs,

    playlists,
    playlistsTotal,
    isLoadingPlaylists,
    loadPlaylists,
    loadMorePlaylists,

    activeTab,
    setActiveTab,

    selectedTracks,
    toggleTrackSelection,
    selectAllTracks,
    clearSelection,
  };
}
