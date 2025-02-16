package slice

import "github.com/lhh-gh/ekit/internal/errs"

// Delete 泛型函数实现切片元素删除
// 参数说明：
//   - src: 原始切片
//   - index: 待删除元素的索引位置
//
// 返回值说明：
//   - []T: 删除指定元素后的新切片
//   - T: 被删除的元素值
//   - error: 错误信息（索引越界时返回）
func Delete[T any](src []T, index int) ([]T, T, error) {
	length := len(src)
	if index < 0 || index >= length {
		var zero T
		return nil, zero, errs.NewErrIndexOutOfRange(length, index)
	}

	// 获取被删除元素的值
	res := src[index]

	// 元素前移操作：从目标索引开始，逐位用后一个元素覆盖前一个
	// 示例：删除索引2的元素 [1,2,3,4] -> [1,2,4,4]
	for i := index; i+1 < length; i++ {
		src[i] = src[i+1]
	}

	// 截取切片长度（丢弃最后一个重复元素）
	// 示例：[1,2,4,4] -> [1,2,4]（length-1）
	src = src[:length-1]

	// 返回新切片、被删除元素和nil错误
	return src, res, nil
}

///需要动态维护有序数据集合
//实现队列/栈等数据结构时的元素移除操作
//处理用户列表、日志记录等需要动态删除的场景
//内存敏感型应用（避免频繁内存分配）
