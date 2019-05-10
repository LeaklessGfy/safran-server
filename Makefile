events:
	http --stream get http://localhost:8888/events?channel="$(ID)"

uploadsmall:
	http -f POST \
		http://localhost:8888/upload \
		experiment='{"reference": "small", "name": "small", "bench": "small", "campaign": "small"}' \
		samples@./csv/smallfile.csv \
		alarms@./csv/event.csv

uploadtest:
	http -f POST \
		http://localhost:8888/upload \
		experiment='{"reference": "test", "name": "test", "bench": "test", "campaign": "test"}' \
		samples@./csv/testfile.csv \
		alarms@./csv/event.csv

simple:
	http get http://localhost:8888/simple
