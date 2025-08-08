# System Utilities (macOS Darwin)

## Standard Unix Commands
The following standard Unix commands are available on macOS Darwin:

### File Operations
```bash
ls          # List directory contents
cd          # Change directory  
pwd         # Print working directory
mkdir       # Create directories
rmdir       # Remove directories
rm          # Remove files
cp          # Copy files
mv          # Move/rename files
find        # Find files and directories
```

### Text Processing
```bash
grep        # Search text patterns
sed         # Stream editor
awk         # Text processing
sort        # Sort lines
uniq        # Remove duplicate lines
wc          # Word/line/character count
head        # Show first lines
tail        # Show last lines
cat         # Display file contents
```

### System Info
```bash
uname       # System information
which       # Locate command
whereis     # Locate binary/source/manual
ps          # Process status
top         # Process monitor
df          # Disk space usage
du          # Directory space usage
```

### Git Commands
```bash
git         # Version control system
git status  # Show working tree status
git add     # Add files to staging
git commit  # Commit changes
git push    # Push to remote
git pull    # Pull from remote
git log     # Show commit history
git diff    # Show differences
```

## macOS-Specific Notes
- Uses BSD versions of some commands (slightly different options than GNU Linux)
- Case-insensitive filesystem by default (HFS+/APFS)
- Standard shell is zsh (since macOS Catalina)
- Package manager: Homebrew (brew command)

## Go-Specific Commands
```bash
go build    # Build Go programs
go run      # Run Go programs
go test     # Run tests
go mod      # Module management
go fmt      # Format Go code
gofmt       # Go formatter
goimports   # Import formatter
```

## Make Commands
All project-specific operations should use the Makefile targets rather than direct system commands where possible.