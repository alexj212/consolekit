#!/bin/bash
# Test script to verify ls @ and cat @ functionality

echo "=== Testing ls @ (list embedded files) ==="
echo "ls @" | go run .

echo ""
echo "=== Testing cat @test.run (read embedded file) ==="
echo "cat @test.run" | go run .

echo ""
echo "=== Testing cat @test1.run (read another embedded file) ==="
echo "cat @test1.run" | go run .

echo ""
echo "=== All tests completed ==="
