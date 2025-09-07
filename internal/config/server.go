package config

type Server struct {
	Port string
}

func newServer() *Server {
	return &Server{
		Port: ":" + getEnv("SERVER_PORT", "8080"),
	}
}
