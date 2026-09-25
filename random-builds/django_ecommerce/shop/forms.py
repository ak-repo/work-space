from django import forms
from django.contrib.auth.forms import UserCreationForm
from django.contrib.auth.models import User


class RegisterForm(UserCreationForm):
    email = forms.EmailField(required=True)

    class Meta:
        model = User
        fields = ['username', 'email', 'password1', 'password2']


class CheckoutForm(forms.Form):
    shipping_name = forms.CharField(max_length=160)
    shipping_address = forms.CharField(widget=forms.Textarea(attrs={'rows': 4}))
