package cluster

type event interface {
	Name() string
	CurrentState() stat
}

type stateEvent struct {
	name  string
	state stat
}

func (se *stateEvent) Name() string {
	return se.name
}

func (se *stateEvent) CurrentState() stat {
	return se.state
}
