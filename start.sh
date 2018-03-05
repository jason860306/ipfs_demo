#!/bin/bash

go test -run TestIpfs -args cli 2>&1 | tee saturn.log
