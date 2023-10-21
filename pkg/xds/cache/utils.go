package cache

func (c *Cache) GetNodeIDsForResource(resourceType, name string) ([]string, error) {
	nodeIDsWithListener := []string{}

	for _, nodeID := range c.nodeIDs {
		cacheSnapshot, err := c.SnapshotCache.GetSnapshot(nodeID)
		if err != nil {
			return nil, err
		}
		listeners := cacheSnapshot.GetResources(resourceType)
		_, ok := listeners[name]
		if ok {
			nodeIDsWithListener = append(nodeIDsWithListener, nodeID)
		}
	}

	return nodeIDsWithListener, nil
}
