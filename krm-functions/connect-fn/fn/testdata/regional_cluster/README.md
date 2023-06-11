# cluster

## Description

This package provides a blueprint for deploying a capi cluster using the capi kind docker templates

The package contains some defaults but can be changed through the kpt pipeline
- pod cidrBlocks: 192.168.0.0/16
- service cidrBlocks: 10.128.0.0/12
- service domain: cluster.local
- kubernetes version: v1.26.3
- workers: 3