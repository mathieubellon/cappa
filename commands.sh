#!/bin/bash
service postgres start
go test -v ./... -coverprofile cover.out