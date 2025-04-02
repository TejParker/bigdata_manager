package api

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/TejParker/bigdata-manager/internal/db"
	"github.com/TejParker/bigdata-manager/pkg/model"
)

// CreateCluster 创建新集群
func CreateCluster(c *gin.Context) {
	var cluster model.Cluster
	if err := c.ShouldBindJSON(&cluster); err != nil {
		ResponseError(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	// 插入集群记录
	query := "INSERT INTO cluster (name, description) VALUES (?, ?)"
	result, err := db.DB.Exec(query, cluster.Name, cluster.Description)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "创建集群失败: "+err.Error())
		return
	}

	// 获取新集群ID
	clusterID, err := result.LastInsertId()
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "获取集群ID失败")
		return
	}

	// 返回成功响应
	cluster.ID = int(clusterID)
	ResponseSuccessWithMessage(c, "集群创建成功", cluster)
}

// GetClusters 获取集群列表
func GetClusters(c *gin.Context) {
	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	// 查询总数
	var total int
	countQuery := "SELECT COUNT(*) FROM cluster"
	err := db.DB.QueryRow(countQuery).Scan(&total)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询集群总数失败")
		return
	}

	// 查询分页数据
	query := "SELECT id, name, description, created_at, updated_at FROM cluster ORDER BY id DESC LIMIT ? OFFSET ?"
	rows, err := db.DB.Query(query, pageSize, offset)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询集群列表失败")
		return
	}
	defer rows.Close()

	// 构造结果集
	var clusters []model.Cluster
	for rows.Next() {
		var cluster model.Cluster
		if err := rows.Scan(&cluster.ID, &cluster.Name, &cluster.Description, &cluster.CreatedAt, &cluster.UpdatedAt); err != nil {
			ResponseError(c, http.StatusInternalServerError, "读取集群数据失败")
			return
		}
		clusters = append(clusters, cluster)
	}

	if err = rows.Err(); err != nil {
		ResponseError(c, http.StatusInternalServerError, "处理集群数据失败")
		return
	}

	// 返回分页结果
	ResponsePageSuccess(c, clusters, total, page, pageSize)
}

// GetClusterById 根据ID获取集群详情
func GetClusterById(c *gin.Context) {
	clusterID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		ResponseError(c, http.StatusBadRequest, "无效的集群ID")
		return
	}

	// 查询集群基本信息
	var cluster model.Cluster
	clusterQuery := "SELECT id, name, description, created_at, updated_at FROM cluster WHERE id = ?"
	err = db.DB.QueryRow(clusterQuery, clusterID).Scan(
		&cluster.ID, &cluster.Name, &cluster.Description, &cluster.CreatedAt, &cluster.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		ResponseError(c, http.StatusNotFound, "集群不存在")
		return
	}
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询集群信息失败")
		return
	}

	// 查询主机数量
	var hostCount int
	hostCountQuery := "SELECT COUNT(*) FROM host WHERE cluster_id = ?"
	err = db.DB.QueryRow(hostCountQuery, clusterID).Scan(&hostCount)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询主机数量失败")
		return
	}

	// 查询服务数量
	var serviceCount int
	serviceCountQuery := "SELECT COUNT(*) FROM service WHERE cluster_id = ?"
	err = db.DB.QueryRow(serviceCountQuery, clusterID).Scan(&serviceCount)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询服务数量失败")
		return
	}

	// 返回结果
	ResponseSuccess(c, gin.H{
		"cluster":       cluster,
		"host_count":    hostCount,
		"service_count": serviceCount,
	})
}

// UpdateCluster 更新集群信息
func UpdateCluster(c *gin.Context) {
	clusterID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		ResponseError(c, http.StatusBadRequest, "无效的集群ID")
		return
	}

	var cluster model.Cluster
	if err := c.ShouldBindJSON(&cluster); err != nil {
		ResponseError(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	// 检查集群是否存在
	var exists bool
	existsQuery := "SELECT EXISTS(SELECT 1 FROM cluster WHERE id = ?)"
	err = db.DB.QueryRow(existsQuery, clusterID).Scan(&exists)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询集群失败")
		return
	}
	if !exists {
		ResponseError(c, http.StatusNotFound, "集群不存在")
		return
	}

	// 更新集群信息
	updateQuery := "UPDATE cluster SET name = ?, description = ? WHERE id = ?"
	_, err = db.DB.Exec(updateQuery, cluster.Name, cluster.Description, clusterID)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "更新集群失败: "+err.Error())
		return
	}

	ResponseSuccessWithMessage(c, "集群更新成功", nil)
}

// DeleteCluster 删除集群
func DeleteCluster(c *gin.Context) {
	clusterID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		ResponseError(c, http.StatusBadRequest, "无效的集群ID")
		return
	}

	// 检查集群是否存在
	var exists bool
	existsQuery := "SELECT EXISTS(SELECT 1 FROM cluster WHERE id = ?)"
	err = db.DB.QueryRow(existsQuery, clusterID).Scan(&exists)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询集群失败")
		return
	}
	if !exists {
		ResponseError(c, http.StatusNotFound, "集群不存在")
		return
	}

	// 检查是否有关联的主机
	var hostCount int
	hostCountQuery := "SELECT COUNT(*) FROM host WHERE cluster_id = ?"
	err = db.DB.QueryRow(hostCountQuery, clusterID).Scan(&hostCount)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询主机数量失败")
		return
	}
	if hostCount > 0 {
		ResponseError(c, http.StatusBadRequest, "集群包含主机，无法删除")
		return
	}

	// 删除集群
	deleteQuery := "DELETE FROM cluster WHERE id = ?"
	_, err = db.DB.Exec(deleteQuery, clusterID)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "删除集群失败: "+err.Error())
		return
	}

	ResponseSuccessWithMessage(c, "集群删除成功", nil)
}

// RegisterClusterRoutes 注册集群相关路由
func RegisterClusterRoutes(router *gin.RouterGroup) {
	router.Use(JWTAuthMiddleware())
	
	// 需要集群查看权限的接口
	viewRouter := router.Group("/")
	viewRouter.Use(PrivilegeMiddleware("VIEW_CLUSTER"))
	{
		viewRouter.GET("/clusters", GetClusters)
		viewRouter.GET("/clusters/:id", GetClusterById)
	}
	
	// 需要集群管理权限的接口
	manageRouter := router.Group("/")
	manageRouter.Use(PrivilegeMiddleware("MANAGE_CLUSTER"))
	{
		manageRouter.POST("/clusters", CreateCluster)
		manageRouter.PUT("/clusters/:id", UpdateCluster)
		manageRouter.DELETE("/clusters/:id", DeleteCluster)
	}
} 