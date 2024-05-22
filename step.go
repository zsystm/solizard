package main

type Step string

const (
	StepChangeContract        Step = "change_contract"
	StepChangeContractAddress Step = "input_contract_address"
	StepSelectMethod          Step = "select_method"
	StepExit                  Step = "exit"
)
