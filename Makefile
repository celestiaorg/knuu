test-basic:
	go test -v ./e2e/basic -timeout 120m

test-basic-file-cache:
	go test -v ./e2e/basic -run=TestFileCache -count=1 -timeout 120m

test-basic-folder-cache:
	go test -v ./e2e/basic -run=TestFolderCache -count=1 -timeout 120m

test-bittwister-packetloss:
	KNUU_TIMEOUT=120m go test -v ./e2e/basic --run=TestBittwister_Packetloss -timeout 60m -count=1

test-bittwister-bandwidth:
	KNUU_TIMEOUT=120m go test -v ./e2e/basic --run=TestBittwister_Bandwidth -timeout 60m -count=1

test-bittwister-latency:
	KNUU_TIMEOUT=120m go test -v ./e2e/basic --run=TestBittwister_Latency -timeout 60m -count=1

test-bittwister-jitter:
	KNUU_TIMEOUT=120m go test -v ./e2e/basic --run=TestBittwister_Jitter -timeout 60m -count=1

test-all:
	KNUU_TIMEOUT=300m go test -v ./e2e/... -timeout 120m

.PHONY: test-all test-basic test-basic-file-cache test-basic-folder-cache test-bittwister-packetloss test-bittwister-bandwidth test-bittwister-latency test-bittwister-jitter

