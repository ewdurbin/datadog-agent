// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package common

import (
	"os"
	"path/filepath"

	//nolint:depguard // creating a logger from a seelog config
	"github.com/cihub/seelog"

	pkgconfigsetup "github.com/DataDog/datadog-agent/pkg/config/setup"
	"github.com/DataDog/datadog-agent/pkg/util/defaultpaths"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"github.com/DataDog/datadog-agent/pkg/util/winutil"
	"github.com/DataDog/datadog-agent/pkg/util/winutil/messagestrings"
)

// ServiceName is the name of the Windows Service the agent runs as
const ServiceName = "DatadogAgent"

func init() {
	_, err := winutil.GetProgramDataDir()
	if err != nil {
		winutil.LogEventViewer(ServiceName, messagestrings.MSG_WARNING_PROGRAMDATA_ERROR, defaultpaths.ConfPath)
	}
}

// EnableLoggingToFile -- set up logging to file
func EnableLoggingToFile() {
	seeConfig := `
<seelog>
	<outputs>
		<rollingfile type="size" filename="c:\\ProgramData\\DataDog\\Logs\\agent.log" maxsize="1000000" maxrolls="2" />
	</outputs>
</seelog>`
	logger, _ := seelog.LoggerFromConfigAsBytes([]byte(seeConfig))
	log.ReplaceLogger(logger)
}

// CheckAndUpgradeConfig checks to see if there's an old datadog.conf, and if
// datadog.yaml is either missing or incomplete (no API key).  If so, upgrade it
func CheckAndUpgradeConfig() error {
	datadogConfPath := filepath.Join(defaultpaths.ConfPath, "datadog.conf")
	if _, err := os.Stat(datadogConfPath); os.IsNotExist(err) {
		log.Debug("Previous config file not found, not upgrading")
		return nil
	}
	pkgconfigsetup.Datadog().AddConfigPath(defaultpaths.ConfPath)
	_, err := pkgconfigsetup.LoadWithoutSecret(pkgconfigsetup.Datadog(), nil)
	if err == nil {
		// was able to read config, check for api key
		if pkgconfigsetup.Datadog().GetString("api_key") != "" {
			log.Debug("Datadog.yaml found, and API key present.  Not upgrading config")
			return nil
		}
	}
	err = ImportConfig(defaultpaths.ConfPath, defaultpaths.ConfPath, false)
	if err != nil {
		winutil.LogEventViewer(ServiceName, messagestrings.MSG_WARN_CONFIGUPGRADE_FAILED, err.Error())
		return err
	}
	return nil
}
