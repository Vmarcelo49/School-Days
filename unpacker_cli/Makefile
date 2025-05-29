# Makefile for School Days GPK Unpacker CLI

CXX = clang++
CXXFLAGS = -std=c++17 -O2 -Wall -Wextra -Izlib/zlib-1.3.1
LIBS = zlib/zlib-1.3.1/libz.a
TARGET = unpacker_cli.exe
TEST_TARGET = test_unpacker.exe
SOURCES = main.cpp filesystem.cpp gpk.cpp
TEST_SOURCES = test.cpp filesystem.cpp gpk.cpp
OBJECTS = $(SOURCES:.cpp=.o)
TEST_OBJECTS = $(TEST_SOURCES:.cpp=.o)

# Default target
all: $(TARGET)

# Build the executable
$(TARGET): $(OBJECTS)
	$(CXX) $(OBJECTS) $(LIBS) -o $(TARGET)

# Build test executable
$(TEST_TARGET): $(TEST_OBJECTS)
	$(CXX) $(TEST_OBJECTS) $(LIBS) -o $(TEST_TARGET)

# Compile source files
%.o: %.cpp
	$(CXX) $(CXXFLAGS) -c $< -o $@

# Run tests
test: $(TEST_TARGET)
	./$(TEST_TARGET)

# Clean build files
clean:
	del /Q *.o *.exe 2>nul || echo "Clean completed"

# Windows specific target
windows: CXX = g++
windows: CXXFLAGS += -static
windows: TARGET = unpacker_cli.exe
windows: $(TARGET)

# Install target (optional)
install: $(TARGET)
	cp $(TARGET) /usr/local/bin/

.PHONY: all clean windows install test

# Dependencies
main.o: main.cpp filesystem.h
filesystem.o: filesystem.cpp filesystem.h gpk.h
gpk.o: gpk.cpp gpk.h
test.o: test.cpp gpk.h
