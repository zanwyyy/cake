package utils

import "github.com/bwmarrin/snowflake"

func InitSnowflake(nodeID int64) error {
	var err error
	node, err = snowflake.NewNode(nodeID)
	return err
}

func NewSessionID() string {
	if node == nil {
		_ = InitSnowflake(1)
	}
	return node.Generate().String()
}
