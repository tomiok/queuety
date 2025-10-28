package server

type Topic struct {
	Name string
}

func NewTopic(name string) Topic {
	return Topic{Name: name}
}

func (t Topic) IsEmpty() bool {
	return t.Name == ""
}
