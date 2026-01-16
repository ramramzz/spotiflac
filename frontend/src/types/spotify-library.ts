export interface SpotifyUserProfile {
  id: string;
  display_name: string;
  email: string;
  image_url: string;
  country: string;
  product: string;
}

export interface SpotifyAuthStatus {
  is_authenticated: boolean;
  user?: SpotifyUserProfile;
}

export interface LibraryTrack {
  id: string;
  spotify_id: string;
  name: string;
  artists: string;
  artist_ids: string[];
  album: string;
  album_id: string;
  album_artist: string;
  duration: string;
  duration_ms: number;
  cover_url: string;
  isrc: string;
  track_number: number;
  disc_number: number;
  total_tracks: number;
  release_date: string;
  added_at: string;
  explicit: boolean;
}

export interface LikedSongsResponse {
  tracks: LibraryTrack[];
  total: number;
}

export interface UserPlaylist {
  id: string;
  name: string;
  description: string;
  owner_name: string;
  owner_id: string;
  cover_url: string;
  track_count: number;
  public: boolean;
  spotify_url: string;
}

export interface UserPlaylistsResponse {
  playlists: UserPlaylist[];
  total: number;
}
