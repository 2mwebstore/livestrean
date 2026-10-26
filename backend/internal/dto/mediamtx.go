package dto

// MediaMTXAuthRequest is what MediaMTX posts to the auth hook for every
// publish and read attempt (authMethod: http).
type MediaMTXAuthRequest struct {
	User     string `json:"user"`
	Password string `json:"password"`
	Token    string `json:"token"`
	IP       string `json:"ip"`
	Action   string `json:"action"` // publish, read, playback, api, metrics, pprof
	Path     string `json:"path"`
	Protocol string `json:"protocol"` // rtmp, webrtc, hls, ...
	ID       string `json:"id"`
	Query    string `json:"query"`
}
