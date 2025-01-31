colima/install: ## Install colima
	brew install colima

colima/start: ## Start colima
	colima start --cpu 4 --memory 8

colima/stop: ## Stop colima
	colima stop
