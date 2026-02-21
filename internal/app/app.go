package app

type App struct {
	provider Provider
}

func New(provider Provider) *App {
	return &App{provider: provider}
}
