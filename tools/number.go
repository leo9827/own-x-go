package tools

import (
    "errors"
    "strconv"
)

func CompareNumber(v1 interface{}, v2 string, f int) (bool, error) {
    if a, ok := v1.(int); ok {
        b, _ := ToInt(v2)
        switch f {
        case 1: //大于
            return a > b, nil
        case 2: //大于等于
            return a >= b, nil
        case 3: //小于
            return a < b, nil
        case 4: //小于等于
            return a <= b, nil
        }
    } else if a, ok := v1.(float32); ok {
        _a := float64(a)
        b, _ := strconv.ParseFloat(v2, 32)
        switch f {
        case 1: //大于
            return _a > b, nil
        case 2: //大于等于
            return _a >= b, nil
        case 3: //小于
            return _a < b, nil
        case 4: //小于等于
            return _a <= b, nil
        }
    } else if a, ok := v1.(float64); ok {
        b, _ := strconv.ParseFloat(v2, 64)
        switch f {
        case 1: //大于
            return a > b, nil
        case 2: //大于等于
            return a >= b, nil
        case 3: //小于
            return a < b, nil
        case 4: //小于等于
            return a <= b, nil
        }
    }
    return false, errors.New("Compare number: Field Type Wrong")
}

// 递增序列函数
func Sequence() func() int {
    i := 0
    return func() int {
        i = i + 1
        return i
    }
}

// 两者取其小
func Min(x, y int) int {
    if x <= y {
        return x
    }
    return y
}

func Btoi(b bool) int {
    if b == true {
        return 1
    }
    return 0
}

func ContainsInt(list []int, item int) bool {
    for _, s := range list {
        if s == item {
            return true
        }
    }
    return false
}
