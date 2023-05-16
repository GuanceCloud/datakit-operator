package admission

import "os"

func javaAgentImage() string {
	return os.Getenv("ENV_DD_JAVA_AGENT_IMAGE")
}

func pythonAgentImage() string {
	return os.Getenv("ENV_DD_PYTHON_AGENT_IMAGE")
}

func jsAgentImage() string {
	return os.Getenv("ENV_DD_JS_AGENT_IMAGE")
}

func logfwdAppImage() string {
	return os.Getenv("ENV_LOGFWD_IMAGE")
}

func ddAgentHost() string {
	return os.Getenv("ENV_DD_AGENT_HOST")
}

func ddTraceAgentPort() string {
	return os.Getenv("ENV_DD_TRACE_AGENT_PORT")
}

func jmxfetchStatsdHost() string {
	return os.Getenv("ENV_DD_JMXFETCH_STATSD_HOST")
}

func jmxfetchStatsdPort() string {
	return os.Getenv("ENV_DD_JMXFETCH_STATSD_PORT")
}
