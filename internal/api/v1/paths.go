package api

const (
	pathsPrefix = "/api/v1"

	pathsUser         = pathsPrefix + "/user"
	pathsUserRegister = pathsUser + "/register"
	pathsUserLogin    = pathsUser + "/login"

	pathsTests               = pathsPrefix + "/tests"
	pathsTestDetails         = pathsTests + "/{scope}"
	pathsTestInstances       = pathsTestDetails + "/instances"
	pathsTestInstanceDetails = pathsTestInstances + "/{name}"
	pathsTestInstanceStatus  = pathsTestInstanceDetails + "/status"
	pathsTestInstanceLogs    = pathsTestInstanceDetails + "/logs"
	pathsTestInstanceStop    = pathsTestInstanceDetails + "/stop"
	pathsTestInstanceExecute = pathsTestInstanceDetails + "/execute"
)
