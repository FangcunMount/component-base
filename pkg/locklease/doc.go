// Package locklease 定义分布式租约锁的通用领域端口。
//
// 本包不依赖 Redis 或其他存储实现，只提供锁规格、锁身份、租约值对象和
// Manager 接口。业务系统可以在自己的包内定义具体锁规格，再由 infra
// adapter 实现该端口。
package locklease
