/*
This is the main application for the Jazigo tool.

Jazigo is a tool written in Go for retrieving configuration for multiple network devices.

See also: https://github.com/udhos/jazigo

Usage:

	jazigo [flag]

Flags are:

	-configPathPrefix string
	      configuration path prefix
	-deviceDelete
	      delete devices specified in stdin
	-deviceImport
	      import devices from stdin
	-deviceList
	      list devices to stdout
	-devicePurge
	      purge devices specified in stdin
	-disableStdoutLog
	      disable logging to stdout
	-logCheckInterval duration
	      interval for checking log file size
	-logMaxFiles int
	      number of log files to keep
	-logMaxSize int
	      size limit for log file
	-logPathPrefix string
	      log path prefix
	-repositoryPath string
	      repository path
	-runOnce
	      exit after scanning all devices once
	-s3region string
	      AWS S3 region
	-webListen string
	      address:port for web UI
	-wwwStaticPath string
	      directory for static www content

By default, jazigo looks for these path prefixes under $JAZIGO_HOME:

	etc/jazigo.conf. (can be overridden with -configPathPrefix)
	log/jazigo.log.  (can be overridden with -logPathPrefix)
	repo             (can be overridden with -repositoryPath)
	www              (can be overridden with -wwwStaticPath)

If $JAZIGO_HOME is not defined, jazigo home defaults to /var/jazigo.

Since root privileges are usually not needed, run the jazigo application as a regular user.
*/
package main
