COVERAGE_REPORT := coverage.txt
COVERAGE_PROFILE := profile.out
COVERAGE_MODE := atomic

test:
	@echo "mode: $(COVERAGE_MODE)" > $(COVERAGE_REPORT); \
	for dir in `find . -name "*.go" | grep -o '.*/' | sort -u | grep -v './fixtures/'`; do \
		go test $$dir -coverprofile=$(COVERAGE_PROFILE) -covermode=$(COVERAGE_MODE); \
		if [ $$? != 0 ]; then \
			exit 2; \
		fi; \
		if [ -f $(COVERAGE_PROFILE) ]; then \
			tail -n +2 $(COVERAGE_PROFILE) >> $(COVERAGE_REPORT); \
			rm $(COVERAGE_PROFILE); \
		fi; \
	done;
clean_example:
	- rm example/**.pb.go
	- rm example/**.proteus.go
reinstall: clean_example
	- rm `which proteus`
	- go install ./cli/proteus
regenerate:
	- make -C example regenerate
