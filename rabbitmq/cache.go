package rabbitmq

// producersMap is a map of producer's uuid. Just hold it in memory.
var producersMap = make(map[string]string)
