package server

type Topic struct {
	name string
}

func NewTopic(name string) Topic {
	return Topic{
		name: name,
	}
}
