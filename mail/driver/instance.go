package driver

func init() {
	drivers = Drivers{
		m: make(map[Type]Driver),
	}
}

type Drivers struct {
	m map[Type]Driver
}

var drivers Drivers

func (t *Drivers) Register(name Type, driver Driver) {
	t.m[name] = driver
}

func Get(name Type) Driver {
	return drivers.m[name]
}

func Register(name Type, driver Driver) {
	drivers.Register(name, driver)
}
