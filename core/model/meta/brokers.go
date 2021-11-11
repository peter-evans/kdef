package meta

import "sort"

// A cluster broker
type Broker struct {
	ID   int32
	Rack string
}

// A slice of Broker
type Brokers []Broker

// A slice of the broker IDs
func (b Brokers) Ids() []int32 {
	ids := make([]int32, len(b))
	for i, broker := range b {
		ids[i] = broker.ID
	}
	return ids
}

// A map of brokers by non-empty broker rack ID
func (b Brokers) BrokersByRack() map[string][]int32 {
	k := make(map[string][]int32)
	for _, broker := range b {
		if len(broker.Rack) > 0 {
			k[broker.Rack] = append(k[broker.Rack], broker.ID)
		}
	}
	return k
}

// A unique, sorted slice of non-empty broker rack IDs
func (b Brokers) Racks() []string {
	bbr := b.BrokersByRack()
	ids := make([]string, len(bbr))
	i := 0
	for id := range bbr {
		ids[i] = id
		i++
	}
	sort.Strings(ids)
	return ids
}

// A map of racks by broker ID
func (b Brokers) RacksByBroker() map[int32]string {
	k := make(map[int32]string)
	for _, broker := range b {
		k[broker.ID] = broker.Rack
	}
	return k
}
