run: frontend api

api:
	go run .

migrate:
	go run github.com/prisma/prisma-client-go db push

clean:
	# Delete DB if exists
	@(sudo -H -u postgres bash -c 'psql -lqt | cut -d \| -f 1 | grep -qw dev') && (sudo -H -u postgres bash -c 'psql -U postgres -c "DROP DATABASE dev;"')
	# Create DB for testing
	-@(sudo -H -u postgres bash -c 'createdb dev')

kill:
	@ps axf | grep "test dev 127.0.0.1" | grep -v grep | awk '{print "sudo kill " $$1}'
	@ps axf | grep "test dev 127.0.0.1" | grep -v grep | awk '{print "sudo kill " $$1}' | bash

frontend:
	cd frontend && npm run build