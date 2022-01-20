make_api:
	god api go --dir . --api wechat.api

make_model:
	god mysql --dsn="root:asdfasdf@tcp(localhost:3306)/wechat_platform" --table="platform,weapp,weapp_audit,pay,pay_order,pay_refund" --dir="./internal/model" --cache