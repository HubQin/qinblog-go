package router

import (
	"github.com/gin-gonic/gin"

	"github.com/qin/qinblog/internal/config"
	adminh "github.com/qin/qinblog/internal/handlers/admin"
	"github.com/qin/qinblog/internal/handlers/web"
	"github.com/qin/qinblog/internal/middleware"
)

// Setup 注册全部路由（对照 routes/web.php + 后台）
func Setup(r *gin.Engine) {
	r.Use(middleware.Sessions(), middleware.Globals(), middleware.VerifyCSRF())

	// 静态资源
	r.Static("/build", "web/public/build")
	r.Static("/images", "web/public/images")
	r.Static("/storage", config.C.UploadPath)
	r.StaticFile("/favicon.ico", "web/public/favicon.ico")

	r.GET("/", web.PostsIndex)
	r.GET("/about", web.About)

	// 认证（等价 Auth::routes(['verify' => true])）
	guest := r.Group("", middleware.Guest())
	{
		guest.GET("/login", web.ShowLogin)
		guest.POST("/login", web.Login)
		guest.GET("/register", web.ShowRegister)
		guest.POST("/register", web.Register)
		guest.GET("/password/reset", web.ShowForgotPassword)
		guest.POST("/password/email", web.SendResetLink)
		guest.GET("/password/reset/:token", web.ShowResetPassword)
		guest.POST("/password/reset", web.ResetPassword)
	}
	r.POST("/logout", web.Logout)
	r.GET("/email/verify/:id", web.EmailVerify)

	auth := r.Group("", middleware.Auth())
	{
		auth.GET("/email/verify", web.EmailVerifyNotice)
		auth.POST("/email/resend", web.EmailResend)

		// 文章创建/编辑/删除
		auth.GET("/posts/create", web.PostsCreate)
		auth.POST("/posts", web.PostsStore)
		auth.GET("/posts/:id/edit", web.PostsEdit)
		auth.PUT("/posts/:id", web.PostsUpdate)
		auth.DELETE("/posts/:id", web.PostsDestroy)
		auth.POST("/posts/upload_post_image", web.UploadPostImage)

		// 评论
		auth.POST("/comments/store", web.CommentsStore)
		auth.POST("/replies/store", web.RepliesStore)
		auth.DELETE("/comments/:id", web.CommentsDestroy)

		// 通知
		auth.GET("/notifications", web.NotificationsIndex)
	}

	// 文章浏览
	r.GET("/posts", web.PostsIndex)
	r.GET("/posts/search", web.PostsSearch)
	r.GET("/posts/:id", web.PostsShow)
	r.GET("/posts/:id/:slug", web.PostsShow)

	// 用户 / 分类 / 标签 / 专题 / 归档
	r.GET("/users/:id", web.UsersShow)
	r.GET("/categories/:id", web.CategoryShow)
	r.GET("/tags/:id", web.TagShow)
	r.GET("/topics", web.TopicsIndex)
	r.GET("/topics/:id", web.TopicShow)
	r.GET("/archives/:year_month", web.ArchiveShow)

	// 第三方登录
	r.GET("/socials/:social_type/redirect", web.SocialRedirect)
	r.GET("/socials/:social_type/callback", web.SocialCallback)

	// 后台管理（users.is_admin 判定）
	admin := r.Group("/admin", middleware.Admin())
	{
		admin.GET("", adminh.Dashboard)
		admin.GET("/posts", adminh.PostsIndex)
		admin.POST("/posts/:id/toggle", adminh.PostToggleShow)
		admin.DELETE("/posts/:id", adminh.PostDestroy)

		admin.GET("/comments", adminh.CommentsIndex)
		admin.POST("/comments/review", adminh.CommentsReview)
		admin.DELETE("/comments/:id", adminh.CommentDestroy)

		admin.GET("/categories", adminh.CategoriesIndex)
		admin.POST("/categories", adminh.CategorySave)
		admin.POST("/categories/:id", adminh.CategorySave)
		admin.DELETE("/categories/:id", adminh.CategoryDestroy)

		admin.GET("/tags", adminh.TagsIndex)
		admin.POST("/tags", adminh.TagSave)
		admin.POST("/tags/:id", adminh.TagSave)
		admin.DELETE("/tags/:id", adminh.TagDestroy)

		admin.GET("/topics", adminh.TopicsIndex)
		admin.POST("/topics", adminh.TopicSave)
		admin.POST("/topics/:id", adminh.TopicSave)
		admin.DELETE("/topics/:id", adminh.TopicDestroy)

		admin.GET("/columns", adminh.ColumnsIndex)
		admin.POST("/columns", adminh.ColumnSave)
		admin.POST("/columns/:id", adminh.ColumnSave)
		admin.DELETE("/columns/:id", adminh.ColumnDestroy)

		admin.GET("/links", adminh.LinksIndex)
		admin.POST("/links", adminh.LinkSave)
		admin.POST("/links/:id", adminh.LinkSave)
		admin.DELETE("/links/:id", adminh.LinkDestroy)

		admin.GET("/users", adminh.UsersIndex)
		admin.GET("/settings", adminh.SettingsShow)
		admin.POST("/settings", adminh.SettingsSave)
	}
}
