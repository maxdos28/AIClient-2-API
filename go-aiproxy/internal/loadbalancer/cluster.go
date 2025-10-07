package loadbalancer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ClusterNode represents a node in the cluster
type ClusterNode struct {
	ID           string    `json:"id"`
	Address      string    `json:"address"`
	IsLeader     bool      `json:"is_leader"`
	LastSeen     time.Time `json:"last_seen"`
	LoadBalancer *LoadBalancer
}

// Cluster manages multiple AI proxy nodes
type Cluster struct {
	mu          sync.RWMutex
	nodeID      string
	nodes       map[string]*ClusterNode
	isLeader    bool
	leaderID    string
	httpClient  *http.Client
	heartbeatInterval time.Duration
	stopChan    chan struct{}
}

// NewCluster creates a new cluster manager
func NewCluster(nodeID, address string) *Cluster {
	return &Cluster{
		nodeID:   nodeID,
		nodes:    make(map[string]*ClusterNode),
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		heartbeatInterval: 10 * time.Second,
		stopChan:         make(chan struct{}),
	}
}

// Join joins a cluster
func (c *Cluster) Join(seedNodes []string) error {
	// Try to connect to seed nodes
	for _, seed := range seedNodes {
		if err := c.connectToNode(seed); err == nil {
			break
		}
	}

	// Start heartbeat
	go c.heartbeatLoop()

	// Start leader election
	go c.leaderElectionLoop()

	return nil
}

// Leave leaves the cluster
func (c *Cluster) Leave() {
	close(c.stopChan)
	
	// Notify other nodes
	c.mu.RLock()
	nodes := make([]*ClusterNode, 0, len(c.nodes))
	for _, node := range c.nodes {
		nodes = append(nodes, node)
	}
	c.mu.RUnlock()

	for _, node := range nodes {
		c.notifyNodeLeave(node.Address)
	}
}

// connectToNode connects to a cluster node
func (c *Cluster) connectToNode(address string) error {
	url := fmt.Sprintf("http://%s/cluster/join", address)
	
	nodeInfo := map[string]interface{}{
		"id":      c.nodeID,
		"address": address,
	}
	
	data, _ := json.Marshal(nodeInfo)
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to join cluster: status %d", resp.StatusCode)
	}

	// Parse cluster info
	var clusterInfo struct {
		Nodes    []ClusterNode `json:"nodes"`
		LeaderID string        `json:"leader_id"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&clusterInfo); err != nil {
		return err
	}

	// Update nodes
	c.mu.Lock()
	for _, node := range clusterInfo.Nodes {
		c.nodes[node.ID] = &node
	}
	c.leaderID = clusterInfo.LeaderID
	c.mu.Unlock()

	return nil
}

// heartbeatLoop sends periodic heartbeats
func (c *Cluster) heartbeatLoop() {
	ticker := time.NewTicker(c.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.sendHeartbeats()
		case <-c.stopChan:
			return
		}
	}
}

// sendHeartbeats sends heartbeats to all nodes
func (c *Cluster) sendHeartbeats() {
	c.mu.RLock()
	nodes := make([]*ClusterNode, 0, len(c.nodes))
	for _, node := range c.nodes {
		if node.ID != c.nodeID {
			nodes = append(nodes, node)
		}
	}
	c.mu.RUnlock()

	for _, node := range nodes {
		go c.sendHeartbeat(node)
	}
}

// sendHeartbeat sends a heartbeat to a specific node
func (c *Cluster) sendHeartbeat(node *ClusterNode) {
	url := fmt.Sprintf("http://%s/cluster/heartbeat", node.Address)
	
	heartbeat := map[string]interface{}{
		"id":        c.nodeID,
		"timestamp": time.Now().Unix(),
	}
	
	data, _ := json.Marshal(heartbeat)
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		// Mark node as potentially failed
		c.markNodeFailed(node.ID)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		c.mu.Lock()
		if n, ok := c.nodes[node.ID]; ok {
			n.LastSeen = time.Now()
		}
		c.mu.Unlock()
	}
}

// leaderElectionLoop performs leader election
func (c *Cluster) leaderElectionLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.checkLeader()
		case <-c.stopChan:
			return
		}
	}
}

// checkLeader checks if current leader is alive and elects new if needed
func (c *Cluster) checkLeader() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if current leader is alive
	if c.leaderID != "" {
		if leader, ok := c.nodes[c.leaderID]; ok {
			if time.Since(leader.LastSeen) < 30*time.Second {
				return // Leader is alive
			}
		}
	}

	// Leader election: node with lowest ID becomes leader
	lowestID := c.nodeID
	for id := range c.nodes {
		if id < lowestID {
			lowestID = id
		}
	}

	c.leaderID = lowestID
	c.isLeader = (lowestID == c.nodeID)

	if c.isLeader {
		c.broadcastLeaderElection()
	}
}

// broadcastLeaderElection broadcasts leader election result
func (c *Cluster) broadcastLeaderElection() {
	// Implementation would broadcast to all nodes
}

// markNodeFailed marks a node as failed
func (c *Cluster) markNodeFailed(nodeID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.nodes[nodeID]; ok {
		delete(c.nodes, nodeID)
		
		// If failed node was leader, trigger new election
		if nodeID == c.leaderID {
			c.leaderID = ""
			go c.checkLeader()
		}
	}
}

// notifyNodeLeave notifies a node about leaving
func (c *Cluster) notifyNodeLeave(address string) {
	url := fmt.Sprintf("http://%s/cluster/leave", address)
	
	data, _ := json.Marshal(map[string]string{
		"id": c.nodeID,
	})
	
	c.httpClient.Post(url, "application/json", bytes.NewReader(data))
}

// RegisterHandlers registers cluster HTTP handlers
func (c *Cluster) RegisterHandlers(router *gin.RouterGroup) {
	router.POST("/join", c.handleJoin)
	router.POST("/heartbeat", c.handleHeartbeat)
	router.POST("/leave", c.handleLeave)
	router.GET("/status", c.handleStatus)
}

// handleJoin handles cluster join requests
func (c *Cluster) handleJoin(ctx *gin.Context) {
	var nodeInfo ClusterNode
	if err := ctx.ShouldBindJSON(&nodeInfo); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.mu.Lock()
	c.nodes[nodeInfo.ID] = &nodeInfo
	nodeInfo.LastSeen = time.Now()
	c.mu.Unlock()

	// Return cluster info
	c.mu.RLock()
	nodes := make([]ClusterNode, 0, len(c.nodes))
	for _, node := range c.nodes {
		nodes = append(nodes, *node)
	}
	leaderID := c.leaderID
	c.mu.RUnlock()

	ctx.JSON(http.StatusOK, gin.H{
		"nodes":     nodes,
		"leader_id": leaderID,
	})
}

// handleHeartbeat handles heartbeat requests
func (c *Cluster) handleHeartbeat(ctx *gin.Context) {
	var heartbeat struct {
		ID        string `json:"id"`
		Timestamp int64  `json:"timestamp"`
	}
	
	if err := ctx.ShouldBindJSON(&heartbeat); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.mu.Lock()
	if node, ok := c.nodes[heartbeat.ID]; ok {
		node.LastSeen = time.Now()
	}
	c.mu.Unlock()

	ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// handleLeave handles leave requests
func (c *Cluster) handleLeave(ctx *gin.Context) {
	var req struct {
		ID string `json:"id"`
	}
	
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.mu.Lock()
	delete(c.nodes, req.ID)
	c.mu.Unlock()

	ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// handleStatus returns cluster status
func (c *Cluster) handleStatus(ctx *gin.Context) {
	c.mu.RLock()
	nodes := make([]ClusterNode, 0, len(c.nodes))
	for _, node := range c.nodes {
		nodes = append(nodes, *node)
	}
	isLeader := c.isLeader
	leaderID := c.leaderID
	c.mu.RUnlock()

	ctx.JSON(http.StatusOK, gin.H{
		"node_id":   c.nodeID,
		"is_leader": isLeader,
		"leader_id": leaderID,
		"nodes":     nodes,
		"total":     len(nodes),
	})
}