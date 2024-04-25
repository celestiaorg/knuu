package utils

import (
	"encoding/json"
	"fmt"
	"strconv"
)

func nodeIdFromStatus(status string) (string, error) {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(status), &result)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling status: %w", err)
	}

	if errorField, ok := result["error"]; ok {
		errorData, ok := errorField.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("error field exists but is not a map[string]interface{}")
		}
		jsonError := &JSONRPCError{}
		if errorCode, ok := errorData["code"].(float64); ok {
			jsonError.Code = int(errorCode)
		}
		if errorMessage, ok := errorData["message"].(string); ok {
			jsonError.Message = errorMessage
		}
		if errorData, ok := errorData["data"].(string); ok {
			jsonError.Data = errorData
		}
		return "", jsonError
	}

	resultData, ok := result["result"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("error getting result from status")
	}
	nodeInfo, ok := resultData["node_info"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("error getting node info from status")
	}
	id, ok := nodeInfo["id"].(string)
	if !ok {
		return "", fmt.Errorf("error getting id from node info")
	}
	return id, nil
}

func latestBlockHeightFromStatus(status string) (int64, error) {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(status), &result)
	if err != nil {
		return 0, fmt.Errorf("error unmarshalling status: %w", err)
	}

	if errorField, ok := result["error"]; ok {
		errorData, ok := errorField.(map[string]interface{})
		if !ok {
			return 0, fmt.Errorf("error field exists but is not a map[string]interface{}")
		}
		jsonError := &JSONRPCError{}
		if errorCode, ok := errorData["code"].(float64); ok {
			jsonError.Code = int(errorCode)
		}
		if errorMessage, ok := errorData["message"].(string); ok {
			jsonError.Message = errorMessage
		}
		if errorData, ok := errorData["data"].(string); ok {
			jsonError.Data = errorData
		}
		return 0, jsonError
	}

	resultData, ok := result["result"].(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("error getting result from status")
	}
	syncInfo, ok := resultData["sync_info"].(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("error getting sync info from status")
	}
	latestBlockHeight, ok := syncInfo["latest_block_height"].(string)
	if !ok {
		return 0, fmt.Errorf("error getting latest block height from sync info")
	}
	latestBlockHeightInt, err := strconv.ParseInt(latestBlockHeight, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("error converting latest block height to int: %w", err)
	}
	return latestBlockHeightInt, nil
}

func chainIdFromStatus(status string) (string, error) {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(status), &result)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling status: %w", err)
	}
	resultData, ok := result["result"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("error getting result from status")
	}
	nodeInfo, ok := resultData["node_info"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("error getting node info from status")
	}
	chainId, ok := nodeInfo["network"].(string)
	if !ok {
		return "", fmt.Errorf("error getting network from node info")
	}
	return chainId, nil
}

func hashFromBlock(block string) (string, error) {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(block), &result)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling block: %w", err)
	}
	resultData, ok := result["result"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("error getting result from block")
	}
	blockId, ok := resultData["block_id"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("error getting block id from block")
	}
	blockHash, ok := blockId["hash"].(string)
	if !ok {
		return "", fmt.Errorf("error getting hash from block id")
	}
	return blockHash, nil
}
