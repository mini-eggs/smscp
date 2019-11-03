dev:
	gin 

deploy:
	cp -r web pkg/handler && \
	now && \
	rm -rf pkg/handler/web

yolo:
	cp -r web pkg/handler && \
	now && now alias tophone.minieggs40.now.sh tophone.evanjon.es && \
	rm -rf pkg/handler/web

	
