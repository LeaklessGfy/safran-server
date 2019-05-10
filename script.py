import requests

API = 'http://localhost:8888'

multipart = {
    'experiment': (None, '{"reference": "test", "name": "test", "bench": "test", "campaign": "test"}'),
    'samples': ('samples', open('csv/smallfile.csv', 'rb')),
    'alarms': ('alarms', open('csv/event.csv', 'rb'))
}

upload = requests.post(API + '/upload', files=multipart)
data = upload.json()

print data['channel']

events = requests.get(API + '/events?channel=' + data['channel'], stream=True)
for line in events.iter_lines():
    if line:
        decoded_line = line.decode('utf-8')
        print(decoded_line)
