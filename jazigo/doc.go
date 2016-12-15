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
*/
package main
