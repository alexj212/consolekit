#!/bin/bash

NEW_TAG="0.0.14"
git tag -s "v${NEW_TAG}" -m "latest version: v${NEW_TAG}"
git push -f origin "v${NEW_TAG}"



