package gndict

type DictGen interface {
	Download() error
	Preprocess() error
	Output() error
}
