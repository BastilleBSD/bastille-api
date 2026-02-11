.PHONY: all
all:
	@echo "Nothing to be done. Please use make install or make uninstall"
.PHONY: install
install:
	@echo
	@echo "Installing BastilleBSD API Server..."
	@echo
	@go build -o /usr/local/bin/bastille-api main.go
	@mkdir -p /usr/local/etc/bastille-api
	@cp -Rv etc/bastille-api/* /usr/local/etc/bastille-api/
	@echo
	@mkdir -p /usr/local/etc/rc.d
	@cp -Rv etc/rc.d/* /usr/local/etc/rc.d/
	@echo
	@echo "This method is for testing & development."
	@echo
	@echo "Please report any issues to https://github.com/BastilleBSD/bastille-ui/issues"
	@echo

.PHONY: uninstall
uninstall:
	@echo
	@echo "Removing BastilleBSD API Server..."
	@echo
	@rm -vf /usr/local/bin/bastille-api
	@echo
	@echo "Removing configuration files..."
	@rm -rvf /usr/local/etc/bastille-api/config.json.sample
	@echo
	@echo "Removing startup script..."
	@rm -vf /usr/local/etc/rc.d/bastille-api
	@echo
	@echo "You may need to manually remove /usr/local/etc/bastille-api/config.json"
	@echo "and /usr/local/share/bastille-api/config.json if they are no longer needed."