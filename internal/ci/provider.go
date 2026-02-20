package ci

type Provider interface {
	GetCurrentCI(zone string) (float64, error)
}
