1. Run containers:
``
docker-compose up
``

2. Install llama AI models:
``
docker exec -it ai_devs3_llama ollama run llama2:7b
``
``
docker exec -it ai_devs3_llama ollama run gemma:2b
``

3. Profit:
``
go run cmd/s01e01/main.go
``
``
go run cmd/s01e02/main.go
``
...
