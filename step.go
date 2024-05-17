package main

type Step string

const (
	StepSelectContract       Step = "select_contract"
	StepInputContractAddress Step = "input_contract_address"
	StepSelectMethod         Step = "select_method"
	StepExit                 Step = "exit"
)
