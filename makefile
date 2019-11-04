dev:
	gin 

deploy:
	cp -r web pkg/handler && \
	now && \
	rm -rf pkg/handler/web

yolo:
	cp -r web pkg/handler && \
	now && now alias smscp.minieggs40.now.sh smscp.evanjon.es && \
	rm -rf pkg/handler/web

	
