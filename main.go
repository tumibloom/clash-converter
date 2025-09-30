package main

func main() {
	InitDb()
	ginEngine := setupRouter()
	err := ginEngine.Run(":8080")
	if err != nil {
		L().Error(err.Error())
	}
}
