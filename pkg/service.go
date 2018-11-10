package pkg

type Service interface {
	Start() error
	Stop() error
}